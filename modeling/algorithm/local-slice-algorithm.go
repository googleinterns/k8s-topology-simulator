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
	"fmt"

	"github.com/googleinterns/k8s-topology-simulator/modeling/types"
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
type LocalSliceAlgorithm struct{}

// CreateSliceGroups creates sliceGroups with 'one local EndpointSliceGroup per
// zone' policy
func (alg LocalSliceAlgorithm) CreateSliceGroups(region types.RegionInfo) (map[string]types.EndpointSliceGroup, error) {
	if region.ZoneDetails == nil {
		return nil, fmt.Errorf("zoneDetail should not be nil")
	}
	sliceGroups := map[string]types.EndpointSliceGroup{}
	// endpointsAvailable stores zones with number of endpoints available
	endpointsAvailable := endpointsList{}
	// endpointsNeeded stores zones with number of endpoints needed
	endpointsNeeded := endpointsList{}
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
		expectedEndpoints := int(zone.NodesRatio * float64(region.TotalEndpoints))
		// deviation: a negative value means need more endpoints from other
		// sliceGroups, a positive value means need give out endpoints to other
		// sliceGroups
		deviation := zone.Endpoints - expectedEndpoints
		weightedEndpoints := types.WeightedEndpoints{Weight: 1}
		if deviation > 0 {
			endpointsAvailable.push(endpointDeviation{name: zoneName, deviation: deviation})
			weightedEndpoints.Number = int(expectedEndpoints)
		} else {
			endpointsNeeded.push(endpointDeviation{name: zoneName, deviation: -deviation})
			weightedEndpoints.Number = zone.Endpoints
		}
		localGroup.Composition[zoneName] = weightedEndpoints
		sliceGroups[zoneName] = localGroup
	}

	err := alg.balanceSliceGroups(&endpointsAvailable, &endpointsNeeded, sliceGroups)
	if err != nil {
		return nil, err
	}
	return sliceGroups, nil
}

// balanceSliceGroups distributes endpoints from zones with extra endpoints to
// EndpointSliceGroups for zones with insufficient endpoints.
func (alg LocalSliceAlgorithm) balanceSliceGroups(endpointsAvailable *endpointsList, endpointsNeeded *endpointsList, sliceGroups map[string]types.EndpointSliceGroup) error {
	for _, receiveZone := range endpointsNeeded.byZone {
		// the available list is empty while there are still endpoints in
		// need. This can happen when the approximation on deviation
		// (calculated above) ends up in asymmetric sums of endpoints
		// available and endpoints in need (in need > available)
		if len(endpointsAvailable.byZone) == 0 {
			// in this case, we do nothing, ignore the extra endpoints needed.
			return nil
		}
		// assign endpoints to the receiveZone
		assignEndpoints(&receiveZone, endpointsAvailable, sliceGroups)
		endpointsNeeded.pop()
	}
	// all endpoints should have been distributed properly. This happens when
	// the sum of approximated available endpoints > sum of approximated
	// endpoints in need
	if len(endpointsAvailable.byZone) != 0 {
		// in this case, we assign those extra endpoints to its local
		// endpointSliceGroup
		for _, extraEndpoints := range endpointsAvailable.byZone {
			originalEndpoints := sliceGroups[extraEndpoints.name].Composition[extraEndpoints.name]
			sliceGroups[extraEndpoints.name].Composition[extraEndpoints.name] = types.WeightedEndpoints{
				Number: originalEndpoints.Number + extraEndpoints.deviation,
				Weight: originalEndpoints.Weight,
			}
			endpointsAvailable.pop()
		}
	}
	return nil
}

// assignEndpoints helps distribute endpoints from rich zones to poor zones
func assignEndpoints(receiveZone *endpointDeviation, endpointsAvailable *endpointsList, sliceGroups map[string]types.EndpointSliceGroup) {
	for index := 0; index < len(endpointsAvailable.byZone); {
		sendZone := endpointsAvailable.byZone[index]
		if sendZone.deviation == receiveZone.deviation {
			sliceGroups[receiveZone.name].Composition[sendZone.name] = types.WeightedEndpoints{Number: sendZone.deviation, Weight: 1}
			endpointsAvailable.pop()
			break
		}
		if sendZone.deviation > receiveZone.deviation {
			sliceGroups[receiveZone.name].Composition[sendZone.name] = types.WeightedEndpoints{Number: receiveZone.deviation, Weight: 1}
			endpointsAvailable.byZone[index].deviation -= receiveZone.deviation
			break
		}
		if sendZone.deviation < receiveZone.deviation {
			sliceGroups[receiveZone.name].Composition[sendZone.name] = types.WeightedEndpoints{Number: sendZone.deviation, Weight: 1}
			receiveZone.deviation -= sendZone.deviation
			endpointsAvailable.pop()
		}
	}
}
