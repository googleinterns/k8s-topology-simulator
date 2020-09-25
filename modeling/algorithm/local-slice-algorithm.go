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

	"github.com/googleinterns/k8s-topology-simulator/modeling/types"
	"k8s.io/klog/v2"
)

// LocalSliceAlgorithm 'borrows' and 'rents' endpoints from other zones to make
// the local EndpointSlice balanced with the incoming traffic (number of nodes
// distribution):
// 1. The proportion of incoming traffic for a zone is calculated based on the
// proportion of nodes or cores in the zone.
// 2. This is compared with the proportion of endpoints in the zone to calculate
// the deviation from the expected number of endpoints in a perfectly balanced
// system.
// 3. EndpointSlices in zones with less endpoints than expected will receive
// endpoints from zones that have more endpoints than expected.
type LocalSliceAlgorithm struct {
	threshold         float64
	startingThreshold int
}

// CreateSliceGroups creates sliceGroups with 'one local EndpointSliceGroup per
// zone' policy
func (alg LocalSliceAlgorithm) CreateSliceGroups(region types.RegionInfo) (map[string]types.EndpointSliceGroup, error) {
	if region.ZoneDetails == nil {
		return nil, fmt.Errorf("zoneDetail should not be nil")
	}
	if region.TotalEndpoints < alg.startingThreshold*len(region.ZoneDetails) {
		return OriginalAlgorithm{}.CreateSliceGroups(region)
	}
	sliceGroups := map[string]types.EndpointSliceGroup{}

	// availablePool consists of zones with endpoints deviation below threshold
	availablePool := ZonePriorityQueue{
		Region:      region,
		SliceGroups: sliceGroups,
	}
	// receiverPool consists of zones with endpoints deviation above threshold
	receiverPool := ZonePriorityQueue{
		Region:          region,
		SliceGroups:     sliceGroups,
		ReceiveEndpoint: true,
	}
	// zonePool consists of all zones, this pool is used to do an extra step of
	// rebalance between zones after each zone has a deviation below threshold
	zonePool := ZonePriorityQueue{
		Region:          region,
		SliceGroups:     sliceGroups,
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

		if zone.Endpoints != 0 {
			localGroup.Composition[zoneName] = types.WeightedEndpoints{Number: zone.Endpoints, Weight: 1}
		}
		sliceGroups[zoneName] = localGroup

		// if this zone would still have a deviation below threshold after
		// giving one endpoint out, it is a qualified contributor
		if alg.validContributor(zoneName, region, sliceGroups) {
			availablePool.ZoneNames = append(availablePool.ZoneNames, zoneName)
		}
		// if this zone has a deviation above threshold, it needs extra
		// endpoints from other zones
		if alg.deviationAboveThreshold(zoneName, region, sliceGroups, 0) {
			receiverPool.ZoneNames = append(receiverPool.ZoneNames, zoneName)
		}
		// add every zone into the zonePool
		zonePool.ZoneNames = append(zonePool.ZoneNames, zoneName)
	}
	succ, err := alg.balanceSliceGroups(&availablePool, &receiverPool, &zonePool, region, sliceGroups)
	if err != nil {
		return nil, err
	}
	if !succ {
		klog.Infof("failed to use local algorithm, switching to original algorithm %+v \n", region)
		return OriginalAlgorithm{}.CreateSliceGroups(region)
	}
	return sliceGroups, nil
}

// balanceSliceGroups distributes endpoints from zones with extra endpoints to
// EndpointSliceGroups for zones with insufficient endpoints.
func (alg LocalSliceAlgorithm) balanceSliceGroups(availablePool *ZonePriorityQueue, receiverPool *ZonePriorityQueue, zonePool *ZonePriorityQueue, region types.RegionInfo, sliceGroups map[string]types.EndpointSliceGroup) (bool, error) {
	heap.Init(availablePool)
	heap.Init(receiverPool)
	// do a first round rebalance, this round aims to get all zones with
	// deviation below threshold
	for receiverPool.Len() > 0 {
		// get the zone with most insufficient endpoints
		receiver := heap.Pop(receiverPool).(string)
		for availablePool.Len() > 0 {
			if !alg.deviationAboveThreshold(receiver, region, sliceGroups, 0) {
				break
			}
			// get the zone with most extra endpoints
			candidate := heap.Pop(availablePool).(string)
			// assign one endpoint from candidate to receiver
			updateSGComposition(sliceGroups[receiver], candidate, 1, 1)
			updateSGComposition(sliceGroups[candidate], candidate, -1, 1)
			// if candidate is still a valid contributor, put it back to the
			// available pool
			if alg.validContributor(candidate, region, sliceGroups) {
				heap.Push(availablePool, candidate)
			}
		}
		// if the receiver still has a deviation above threshold while there is
		// no zones can give endpoints out, downgrade to other algorithm
		if alg.deviationAboveThreshold(receiver, region, sliceGroups, 0) {
			return false, nil
		}
	}
	// rebalance endpoints to reduce mean deviation at the cost of in-zone
	// traffic
	// +optional
	heap.Init(zonePool)
	for availablePool.Len() > 0 {
		// get the zone with most extra endpoints
		candidate := heap.Pop(availablePool).(string)
		deviation, ok := getEndpointsDeviation(region, sliceGroups, candidate)
		if !ok {
			klog.Warningf("get deviation of %s failed", candidate)
			continue
		}
		// if this zone has less than 1 endpoint overflowed compared to expected
		// number, stop the rebalance
		if deviation < 1 {
			break
		}
		// assign extra endpoints to zones with endpoints fewer than expected
		// number
		for zonePool.Len() > 0 {
			// zonePool will always be non-empty
			// get the zone with most insufficient endpoints
			receiver := heap.Pop(zonePool).(string)
			receiverDeviation, ok := getEndpointsDeviation(region, sliceGroups, receiver)
			if !ok {
				klog.Warningf("get deviation of %s failed", receiver)
				continue
			}
			// if this zone has number of endpoints >= floor of expected number,
			// stop the rebalance
			if receiverDeviation > -1 {
				return true, nil
			}
			// assign endpoints from candidate to receiver until one of them
			// hits the boundary
			for deviation >= 1 && receiverDeviation <= -1 {
				updateSGComposition(sliceGroups[receiver], candidate, 1, 1)
				updateSGComposition(sliceGroups[candidate], candidate, -1, 1)
				deviation--
				receiverDeviation++
			}
			heap.Push(zonePool, receiver)
			if deviation < 1 {
				break
			}
		}
	}
	return true, nil
}

// detect whether a zone is valid to contribute endpoints to other zones
func (alg LocalSliceAlgorithm) validContributor(zoneName string, region types.RegionInfo, sliceGroups map[string]types.EndpointSliceGroup) bool {
	// if the sliceGroup has no local composition, it is not a valid contributor
	if len(sliceGroups[zoneName].Composition) == 0 || sliceGroups[zoneName].NumberOfEndpoints() <= 1 {
		return false
	}
	return !alg.deviationAboveThreshold(zoneName, region, sliceGroups, -1)
}

// check if endpoints in a zone have invalid deviation
// zero delta : if current state is above threshold
// negative delta: after giving out abs(delta) endpoints, if it is still above
// threshold
// positive delta: after receiving delta endpoints, if it is still above
// threshold
func (alg LocalSliceAlgorithm) deviationAboveThreshold(zone string, region types.RegionInfo, sliceGroups map[string]types.EndpointSliceGroup, delta int) bool {
	expectedEndpoints := float64(region.TotalEndpoints) * region.ZoneDetails[zone].NodesRatio
	trafficDeviation := expectedEndpoints/float64(sliceGroups[zone].NumberOfEndpoints()+delta) - 1
	return trafficDeviation >= alg.threshold
}
