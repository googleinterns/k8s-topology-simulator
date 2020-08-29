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
	"sort"

	"github.com/googleinterns/k8s-topology-simulator/modeling/types"
	"k8s.io/klog/v2"
)

// LocalSharedSliceAlgorithm is one variation of LocalSliceAlgorithm which
// 'borrows' and 'rents' endpoints from other zones to make the local
// EndpointSliceGroup balanced with the incoming traffic (number of nodes
// distribution). This variation deals with failed corner cases by sharing
// endpoints to zones that have no endpoints.
type LocalSharedSliceAlgorithm struct {
	// threshold for max deviation allowed for endpoints
	threshold float64
}

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
	// traverse the map by name order
	zoneNames := sortZoneByNames(region.ZoneDetails)
	for _, zoneName := range zoneNames {
		zone := region.ZoneDetails[zoneName]
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

		// if deviation > 0 and has more than 1 endpoints in its local
		// sliceGroup, this zone is a qualified candidate for availablePool that
		// contributes endpoints to other zones
		if alg.validContributor(zoneName, region, sliceGroups) {
			availablePool.ZoneNames = append(availablePool.ZoneNames, zoneName)
		}
		receiverPool.ZoneNames = append(receiverPool.ZoneNames, zoneName)

		// deviation = -0.xx will end up with 0, omit those cases
		if deviation <= -1 {
			endpointsNeeded.push(endpointDeviation{name: zoneName, deviation: int(-deviation)})
		}
	}
	availablePool.SliceGroups = sliceGroups
	receiverPool.SliceGroups = sliceGroups

	succ, err := alg.balanceSliceGroups(&endpointsNeeded, &endpointsNeededUrgent, region, sliceGroups, &availablePool, &receiverPool)
	if err != nil {
		return nil, err
	}
	if !succ {
		klog.Infof("failed to use local shared algorithm, switching to shared global algorithm %+v \n", region)
		return OriginalAlgorithm{}.CreateSliceGroups(region)
	}
	return sliceGroups, nil
}

// balanceSliceGroups distributes endpoints from zones with extra endpoints to
// EndpointSliceGroups for zones with insufficient endpoints.
func (alg LocalSharedSliceAlgorithm) balanceSliceGroups(endpointsNeeded *endpointsList, endpointsNeededUrgent *endpointsList, region types.RegionInfo, sliceGroups map[string]types.EndpointSliceGroup, availablePool *ZonePriorityQueue, receiverPool *ZonePriorityQueue) (bool, error) {
	heap.Init(availablePool)
	// merge one sharedSG that zones in the urgent list will consume
	mergedSG := types.EndpointSliceGroup{Composition: map[string]types.WeightedEndpoints{}, ZoneTrafficWeights: map[string]float64{}}
	// merged deviation for urgent zones, this stores the deviation value after
	// approximation from float to int
	mergedED := endpointDeviation{name: "merged"}
	// accumulate the float deviation of every urgent zone, this is an actual
	// value of sum(expectedEndpoints)
	expectedEndpointsMerged := 0.0
	for _, urgentZone := range endpointsNeededUrgent.byZone {
		mergedED.name += "-" + urgentZone.name
		expectedEndpointsMerged += (float64(urgentZone.deviation) * urgentZone.weight)
		mergedSG.ZoneTrafficWeights[urgentZone.name] = 1
		endpointsNeededUrgent.pop()
	}
	if expectedEndpointsMerged >= 1 {
		// workaround with internal float precision lost, this precision lost
		// happens with constant numeric assigned to a float64 variable. One
		// real case: 1 0, 6 0, 7 3 (nodes first), the merged SG for zone1 and
		// zone2 will end up with expected endpoints == 1.4999998. This
		// workaround could fix the wrong value both y.x0000001 or y.(x-1)999998
		// to the desired y.x. This workaround is based on the assumption that
		// the precision lost will only happen in a very latter decimal
		// position.
		expectedEndpointsMerged = math.Ceil(expectedEndpointsMerged*1000) / 1000
		mergedED.deviation = int(math.Round(expectedEndpointsMerged))
	} else {
		// take the ceil, if the sum of expected endpoints < 1, we have to make
		// sure there is at least one endpoint in this shared SG (avoid 0.x
		// rounded down to 0)
		mergedED.deviation = int(math.Ceil(expectedEndpointsMerged))
	}
	mergedSG.Label = mergedED.name
	if expectedEndpointsMerged != 0 {
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
		// expected and has more than one endpoints in its local sliceGroup.
		// candidate is guaranteed to have a local owned SG, omit the second
		// returned value
		if alg.validContributor(candidate, region, sliceGroups) {
			heap.Push(availablePool, candidate)
		}

		receiveZone.deviation--
		if receiveZone.deviation == 0 {
			endpointsNeeded.pop()
		} else {
			endpointsNeeded.byZone[index] = receiveZone
		}
	}
	heap.Init(receiverPool)
	succ := alg.keepDeviationBelowThreshold(availablePool, receiverPool)
	if !succ {
		return succ, nil
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
		deviation, ok := getEndpointsDeviation(region, sliceGroups, candidate)
		if !ok {
			// this should never happen, since every candidate in the
			// availablePool should have a local SG.
			return false, fmt.Errorf("unexpcted nil error in sliceGroups while getting deviation for %s candidate", candidate)
		}
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
		if alg.validContributor(candidate, region, sliceGroups) {
			heap.Push(availablePool, candidate)
		}
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

// helper function helps to keep all the endpoints with a traffic load deviation
// less than threshold, return false if it can't.
func (alg LocalSharedSliceAlgorithm) keepDeviationBelowThreshold(availablePool *ZonePriorityQueue, receiverPool *ZonePriorityQueue) bool {
	region := availablePool.Region
	sliceGroups := availablePool.SliceGroups
	// get zones with deviation >= threshold
	var urgentZones []string
	for receiverPool.Len() > 0 {
		receiveZone := receiverPool.ZoneNames[0]
		if !alg.deviationAboveThreshold(receiveZone, region, sliceGroups, 0) {
			// if the deviation of the first element in receiverPool is below
			// threshold, it means all the elements in the receiverPool have a
			// deviation below threshold. receiverPool is a priority-queue with
			// max deviation first.
			break
		}
		urgentZones = append(urgentZones, receiveZone)
		heap.Pop(receiverPool)
	}

	// get extra endpoints from zones that have more endpoints than the ceiling
	// of their expected endpoints.
	extraEndpoints := map[string]int{}
	extraEndpointsNumber := 0
	if len(urgentZones) > 0 {
		// if extraEndpointsNumber >= len(urgentZones), we just assign one
		// endpoint to each urgent zone, their deviation will go below threshold
		for extraEndpointsNumber < len(urgentZones) {
			if availablePool.Len() > 0 {
				candidate := availablePool.ZoneNames[0]
				deviation, ok := getEndpointsDeviation(region, sliceGroups, candidate)
				if !ok {
					// this should never happen, since every candidate in the
					// availablePool should have a local SG.
					klog.Errorf("unexpcted nil error in sliceGroups while getting deviation for %s candidate", candidate)
					return false
				}
				// zones have absolute extra endpoints directly give them out
				if deviation >= 1 {
					_ = heap.Pop(availablePool)
					extraEndpoints[candidate]++
					extraEndpointsNumber++
					// valid contributor check
					updateSGComposition(sliceGroups[candidate], candidate, -1, 1)
					if alg.validContributor(candidate, region, sliceGroups) {
						heap.Push(availablePool, candidate)
					}
					continue
				}
			}
			// check if current extra endpoints are able to make a shared
			// sliceGroup with deviation < threshold
			if alg.sufficientExtraEndpointsForSharedSlice(urgentZones, region, sliceGroups, extraEndpointsNumber) {
				alg.createSharedSlice(urgentZones, extraEndpoints, sliceGroups)
				return true
			}
			// if current extra endpoints are not enough, we get more endpoints
			// from zones have relatively extra endpoints (different from
			// absolute above). In this case, zones have 5 endpoints expected
			// 4.x endpoints, as long as after giving out one endpoint, its
			// deviation is still below threhold, we ask these zones to give out
			// endpoints
			if alg.getExtraEndpointsForSharedSlice(availablePool, extraEndpoints, urgentZones) {
				alg.createSharedSlice(urgentZones, extraEndpoints, sliceGroups)
				return true
			}
			return false
		}
		// sort zone names to deterministically traverse the map
		var zoneNames []string
		for zone := range extraEndpoints {
			zoneNames = append(zoneNames, zone)
		}
		sort.Strings(zoneNames)
		// assign endpoints to zone needed one by one
		for _, zone := range zoneNames {
			for _, urgentZone := range urgentZones {
				updateSGComposition(sliceGroups[urgentZone], zone, 1, 1)
				urgentZones = urgentZones[1:]
				extraEndpoints[zone]--
				if extraEndpoints[zone] == 0 {
					break
				}
			}
		}
	}
	return true
}

// detect whether a zone is valid to contribute endpoints to other zones
func (alg LocalSharedSliceAlgorithm) validContributor(zoneName string, region types.RegionInfo, sliceGroups map[string]types.EndpointSliceGroup) bool {
	// if the sliceGroup has no local composition, it is not a valid contributor
	if sliceGroups[zoneName].Composition == nil || sliceGroups[zoneName].NumberOfEndpoints() == 1 {
		return false
	}
	return !alg.deviationAboveThreshold(zoneName, region, sliceGroups, -1)
}

// check if endpoints in receiveZone have invalid deviation
func (alg LocalSharedSliceAlgorithm) deviationAboveThreshold(receiveZone string, region types.RegionInfo, sliceGroups map[string]types.EndpointSliceGroup, delta int) bool {
	expectedEndpoints := float64(region.TotalEndpoints) * region.ZoneDetails[receiveZone].NodesRatio
	trafficDeviation := expectedEndpoints/float64(sliceGroups[receiveZone].NumberOfEndpoints()+delta) - 1
	return trafficDeviation >= alg.threshold
}

// check if endpoints in a shared sliceGroup could be able to achieve deviation
// less than threshold
func (alg LocalSharedSliceAlgorithm) sufficientExtraEndpointsForSharedSlice(urgentZones []string, region types.RegionInfo, sliceGroups map[string]types.EndpointSliceGroup, extraEndpoints int) bool {
	trafficLoad := 0.0
	totalEndpoints := extraEndpoints
	for _, urgentZone := range urgentZones {
		totalEndpoints += sliceGroups[urgentZone].NumberOfEndpoints()
	}
	// traffic load = sum(exptected endpoints) / total endpoints in the shared
	// sliceGroup
	for _, urgentZone := range urgentZones {
		expectedEP := float64(region.TotalEndpoints) * region.ZoneDetails[urgentZone].NodesRatio
		trafficLoad += expectedEP / float64(totalEndpoints)
	}
	return trafficLoad-1 < alg.threshold
}

// create a shared sliceGroup for urgent zones that have a deviation
// greater/equal to threshold
func (alg LocalSharedSliceAlgorithm) createSharedSlice(urgentZones []string, extraEndpoints map[string]int, sliceGroups map[string]types.EndpointSliceGroup) {
	sharedLabel := "shared"
	sharedSG := types.EndpointSliceGroup{Composition: map[string]types.WeightedEndpoints{}, ZoneTrafficWeights: map[string]float64{}}
	for _, urgentZone := range urgentZones {
		sharedLabel += fmt.Sprintf("-%s", urgentZone)
		for zone, contribution := range sliceGroups[urgentZone].Composition {
			// urgent zones are contributing all of their endpoints to the
			// shared SG.
			updateSGComposition(sharedSG, zone, contribution.Number, contribution.Weight)
		}
		sharedSG.ZoneTrafficWeights[urgentZone] = 1
		delete(sliceGroups, urgentZone)
	}
	for zone, number := range extraEndpoints {
		updateSGComposition(sharedSG, zone, number, 1)
	}
	sharedSG.Label = sharedLabel
	sliceGroups[sharedLabel] = sharedSG
}

// getExtraEndpointsForSharedSlice attempts to get extra endpoints that could be
// used for a shared slice to get deviation below threshold. Returns false if it
// can't.
// Previously we only ask zones to give out endpoints before they reach the
// ceiling of their expected endpoints. In this function, we ask zones to give
// out endpoints as long as their deviations are less than threshold.
func (alg LocalSharedSliceAlgorithm) getExtraEndpointsForSharedSlice(availablePool *ZonePriorityQueue, extraEndpoints map[string]int, urgentZones []string) bool {
	sliceGroups := availablePool.SliceGroups
	region := availablePool.Region
	// total number of extra endpoints, this value is used to check if it's
	// enough to make a shared SG that has a deviation less than threshold
	extraEndpointsNumber := 0
	for _, num := range extraEndpoints {
		extraEndpointsNumber += num
	}
	// rebalance endpoints until deviation of the shared SG is below threshold
	// or nothing left in availablePool
	for !alg.sufficientExtraEndpointsForSharedSlice(urgentZones, availablePool.Region, sliceGroups, extraEndpointsNumber) {
		// no more endpoints available from other zones, this algorithm returns
		// fail
		if availablePool.Len() == 0 {
			return false
		}
		candidate := heap.Pop(availablePool).(string)
		updateSGComposition(sliceGroups[candidate], candidate, -1, 1)
		// if the candidate is still a valid contributor after giving out one
		// endpoint, push it back to the available queue
		if alg.validContributor(candidate, region, sliceGroups) {
			heap.Push(availablePool, candidate)
		}
		extraEndpointsNumber++
		extraEndpoints[candidate]++
	}
	return true
}
