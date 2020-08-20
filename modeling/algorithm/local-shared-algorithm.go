/*
Copyright 2020 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package algorithm

import (
	"container/heap"
	"fmt"
	"math"

	"github.com/googleinterns/k8s-topology-simulator/modeling/types"
	"k8s.io/klog/v2"
)

// LocalSharedSliceAlgorithm is one variation of LocalSliceAlgorithm which
// 'borrows' and 'rents' endpoints from other zones to make the local
// EndpointSliceGroup balanced with the incoming traffic (number of nodes
// distribution). This variation deals with failed corner cases by sharing
// endpoints to zones that have no endpoints.
type LocalSharedSliceAlgorithm struct{}

// CreateSliceGroups creates sliceGroups with 'one local EndpointSliceGroup per
// zone' policy. Zones with no endpoints allocated will be treated as a whole
// that shares a shared-SG.
func (alg LocalSharedSliceAlgorithm) CreateSliceGroups(region types.RegionInfo) (map[string]types.EndpointSliceGroup, error) {
	if region.ZoneDetails == nil {
		return nil, fmt.Errorf("zoneDetail should not be nil")
	}
	// if number of total endpoints < number of zones, use original algorithm
	// instead. This algorithm itself can handle some of these special corner
	// cases but performs poorly at small scale corner cases, so using the
	// original algorithm seems a better solution in terms of performance and
	// simplicity.
	if region.TotalEndpoints < len(region.ZoneDetails) {
		return OriginalAlgorithm{}.CreateSliceGroups(region)
	}
	sliceGroups := map[string]types.EndpointSliceGroup{}
	// endpointsNeeded stores zones with number of endpoints needed
	endpointsNeeded := endpointsList{}
	// endpointsNeededUrgent stores high priority zones with number of endpoints
	// needed. high priority zones: zones have no endpoints
	endpointsNeededUrgent := endpointsList{}
	// availablePool is used to contribute endpoints shared with zones when there
	// are not enough endpoints in the available list.
	availablePool := ZonePriorityQueue{
		Region: region,
	}
	// receiverPool contains all zones. This is used to receive extra endpoints
	// when there are not enough endpoints in the needed list.
	receiverPool := ZonePriorityQueue{
		Region:          region,
		ReceiveEndpoint: true,
	}
	for zoneName, zone := range region.ZoneDetails {
		var localGroup types.EndpointSliceGroup
		localGroup.Label = zoneName
		// this local sliceGroup should only receive traffic from current zone,
		// this map tracks weights of traffic from different zones to this
		// sliceGroup
		localGroup.ZoneTrafficWeights = map[string]float64{zoneName: 1.0}
		// this map stores composition of this sliceGroup, zoneName - number of
		// endpoints from zoneName
		localGroup.Composition = map[string]types.WeightedEndpoints{}

		// calculate expected number of endpoints based on the proportion of
		// nodes in this zone
		expectedEndpoints := zone.NodesRatio * float64(region.TotalEndpoints)
		// deviation: a negative value means need more endpoints from other
		// sliceGroups, a positive value means need give out endpoints to other
		// sliceGroups
		deviation := float64(zone.Endpoints) - expectedEndpoints
		// if deviation > 0, this zone is a qualified candidate for
		// availablePool that contributes endpoints to other zones
		if deviation > 0 {
			availablePool.ZoneNames = append(availablePool.ZoneNames, zoneName)
		}
		receiverPool.ZoneNames = append(receiverPool.ZoneNames, zoneName)
		// merge all the zones with no endpoints into a shared slice group
		if zone.Endpoints == 0 {
			// this is a form used to accurately represent the deviation in
			// float, floatDeviation = 1 * abs(deviation). If we do the
			// approximate here, it could lead to a large accuracy lose with
			// the accumulated approximated value. We keep the accurate value
			// for these zones and convert to integer after merge.
			endpointsNeededUrgent.push(endpointDeviation{name: zoneName, deviation: 1, weight: -deviation})
			continue
		}
		localGroup.Composition[zoneName] = types.WeightedEndpoints{Number: zone.Endpoints, Weight: 1}
		sliceGroups[zoneName] = localGroup

		// deviation = -0.xx will end up with 0, omit those cases
		if deviation <= -1 {
			endpointsNeeded.push(endpointDeviation{name: zoneName, deviation: int(-deviation)})
		}
	}
	availablePool.SliceGroups = sliceGroups
	receiverPool.SliceGroups = sliceGroups

	succ, err := alg.balanceSliceGroups(&endpointsNeeded, &endpointsNeededUrgent, sliceGroups, &availablePool, &receiverPool)
	if !succ {
		klog.Infoln("failed to use local algorithm, switching to shared global algorithm")
		sharedAlg := SharedGlobalAlgorithm{SharedGlobalAlgorithmCore{globalWeight: 1, globalThreshold: 100}}
		return sharedAlg.CreateSliceGroups(region)
	}
	if err != nil {
		return nil, err
	}
	return sliceGroups, nil
}

// balanceSliceGroups distributes endpoints from zones with extra endpoints to
// EndpointSliceGroups for zones with insufficient endpoints.
func (alg LocalSharedSliceAlgorithm) balanceSliceGroups(endpointsNeeded *endpointsList, endpointsNeededUrgent *endpointsList, sliceGroups map[string]types.EndpointSliceGroup, availablePool *ZonePriorityQueue, receiverPool *ZonePriorityQueue) (bool, error) {
	heap.Init(availablePool)
	heap.Init(receiverPool)
	// merge one sharedSG that zones in the urgent list will consume
	mergedSG := types.EndpointSliceGroup{Composition: map[string]types.WeightedEndpoints{}, ZoneTrafficWeights: map[string]float64{}}
	// merged deviation for urgent zones, this stores the deviation value after
	// approximation from float to int
	mergedED := endpointDeviation{name: "merged"}
	// accumulate the float deviation of every urgent zone, this is an actual
	// value of sum(deviation)
	mergedDeviation := 0.0
	for _, urgentZone := range endpointsNeededUrgent.byZone {
		mergedED.name += "-" + urgentZone.name
		mergedDeviation += (float64(urgentZone.deviation) * urgentZone.weight)
		mergedSG.ZoneTrafficWeights[urgentZone.name] = 1
		endpointsNeededUrgent.pop()
	}
	// take the ceil, if the deviation > 0, we have to make sure there is at
	// least one endpoint in this shared SG (avoid 0.x ending up with 0)
	mergedED.deviation = int(math.Ceil(mergedDeviation))
	mergedSG.Label = mergedED.name
	if mergedDeviation != 0 {
		sliceGroups[mergedSG.Label] = mergedSG
		endpointsNeeded.pushFront(mergedED)
	}
	// assign extra endpoints to zones/SG needed
	for index := 0; index < len(endpointsNeeded.byZone); {
		receiveZone := endpointsNeeded.byZone[index]
		if availablePool.Len() == 0 {
			// if no zones can give endpoints to needed zones, this
			// variation can't work with this input, we handle the input with
			// other algorithms instead.
			return false, nil
		}
		candidate := heap.Pop(availablePool).(string)
		// give one endpoint out
		updateSGComposition(sliceGroups[candidate], candidate, -1, 1)
		// get the one endpoint
		updateSGComposition(sliceGroups[receiveZone.name], candidate, 1, 1)

		// remain a potential provider as long as it has more endpoints than
		// expected. candidate is guaranteed to have a local owned SG, omit the
		// second returned value
		deviation, _ := getEndpointsDeviation(availablePool.Region, availablePool.SliceGroups, candidate)
		if deviation > 0 {
			heap.Push(availablePool, candidate)
		}

		receiveZone.deviation--
		if receiveZone.deviation == 0 {
			endpointsNeeded.pop()
		} else {
			endpointsNeeded.byZone[index] = receiveZone
		}
	}
	// assign extra endpoints to appropriate zones
	// i.e. (nodes, endpoints), (1 3, 2 2, 2 2)
	// the second and third zone will not require endpoints based on floor
	// approximation (2.8 -> 2). But the 1st zone has too many endpoints, it
	// should give one out.
	for availablePool.Len() > 0 {
		candidate := heap.Pop(availablePool).(string)
		// candidate is guaranteed to have a local owned SG, omit the second
		// returned value
		deviation, _ := getEndpointsDeviation(availablePool.Region, availablePool.SliceGroups, candidate)
		if deviation < 1 {
			break
		}
		// if candidate zone has at least one extra endpoints than it
		// expects, it should give that endpoint out to a zone that needs
		// endpoints from other zones.
		receiveZone := heap.Pop(receiverPool).(string)
		updateSGComposition(sliceGroups[receiveZone], candidate, 1, 1)
		heap.Push(receiverPool, receiveZone)

		updateSGComposition(sliceGroups[candidate], candidate, -1, 1)
		heap.Push(availablePool, candidate)
	}
	return true, nil
}

// helper function help calculate the deviation between locally owned endpoints
// and expected endpoints.
func getEndpointsDeviation(region types.RegionInfo, sliceGroups map[string]types.EndpointSliceGroup, zone string) (float64, bool) {
	expectedEndpoints := float64(region.TotalEndpoints) * region.ZoneDetails[zone].NodesRatio
	sliceGroup, ok := sliceGroups[zone]
	if !ok {
		return 0.0, false
	}
	return float64(sliceGroup.Composition[zone].Number) - expectedEndpoints, true
}

// helper function to update composition in ESG
func updateSGComposition(sliceGroup types.EndpointSliceGroup, zone string, delta int, weight float64) {
	weightedComp := sliceGroup.Composition[zone]
	weightedComp.Number += delta
	weightedComp.Weight = weight
	sliceGroup.Composition[zone] = weightedComp
}
