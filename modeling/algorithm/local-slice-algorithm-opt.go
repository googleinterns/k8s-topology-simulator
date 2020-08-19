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
	"math"

	"github.com/googleinterns/k8s-topology-simulator/modeling/types"
)

// LocalSliceAlgorithmOpt is a variation of LocalSliceAlgorithm which 'borrows'
// and 'rents' endpoints from other zones to make the local EndpointSlice
// balanced with the incoming traffic (number of nodes distribution). This
// variation distributes extra endpoints available after local-slice
// distribution to a global SG with a lower weight that every zone can reach.
type LocalSliceAlgorithmOpt struct{}

// CreateSliceGroups creates sliceGroups with 'one local EndpointSliceGroup per
// zone' policy
func (alg LocalSliceAlgorithmOpt) CreateSliceGroups(region types.RegionInfo) (map[string]types.EndpointSliceGroup, error) {
	if region.ZoneDetails == nil {
		return nil, fmt.Errorf("zoneDetail should not be nil")
	}
	sliceGroups := map[string]types.EndpointSliceGroup{}
	// endpointsAvailable stores zones with number of endpoints available
	endpointsAvailable := endpointsList{}
	// endpointsNeeded stores zones with number of endpoints needed
	endpointsNeeded := endpointsList{}
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
		weightedEndpoints := types.WeightedEndpoints{Weight: 1}
		if deviation > 0 {
			endpointsAvailable.push(endpointDeviation{name: zoneName, deviation: int(math.Ceil(deviation))})
			weightedEndpoints.Number = int(expectedEndpoints)
		} else if deviation < 0 {
			endpointsNeeded.push(endpointDeviation{name: zoneName, deviation: int(-deviation)})
			weightedEndpoints.Number = zone.Endpoints
		} else {
			weightedEndpoints.Number = zone.Endpoints
		}
		localGroup.Composition[zoneName] = weightedEndpoints
		sliceGroups[zoneName] = localGroup
	}

	err := alg.balanceSliceGroups(region, &endpointsAvailable, &endpointsNeeded, sliceGroups)
	if err != nil {
		return nil, err
	}
	return sliceGroups, nil
}

// balanceSliceGroups distributes endpoints from zones with extra endpoints to
// EndpointSliceGroups for zones with insufficient endpoints.
func (alg LocalSliceAlgorithmOpt) balanceSliceGroups(region types.RegionInfo, endpointsAvailable *endpointsList, endpointsNeeded *endpointsList, sliceGroups map[string]types.EndpointSliceGroup) error {
	for _, receiveZone := range endpointsNeeded.byZone {
		// the available list is empty while there are still endpoints in
		// need. This can happen when the approximation on deviation
		// (calculated above) ends up in asymmetric sums of endpoints
		// available and endpoints in need (in need > available)
		if len(endpointsAvailable.byZone) == 0 {
			// in this case, we do nothing, ignore the extra endpoints needed.
			// return errors.New("unexpected endpoints in need")
			return nil
		}
		// same as original local algorithm assignment
		assignEndpoints(&receiveZone, endpointsAvailable, sliceGroups)
		endpointsNeeded.pop()
	}
	// This happens when the sum of approximated available endpoints > sum of
	// approximated endpoints in need
	if len(endpointsAvailable.byZone) != 0 {
		// in this case, we assign those extra endpoints to a global
		// endpointSliceGroup
		globalSG := types.EndpointSliceGroup{Label: "global",
			Composition:        map[string]types.WeightedEndpoints{},
			ZoneTrafficWeights: map[string]float64{},
		}
		for zone := range region.ZoneDetails {
			globalSG.ZoneTrafficWeights[zone] = 1 / float64(len(region.ZoneDetails))
		}
		for _, extraEndpoints := range endpointsAvailable.byZone {
			globalSG.Composition[extraEndpoints.name] = types.WeightedEndpoints{Number: extraEndpoints.deviation, Weight: 1.0}
			endpointsAvailable.pop()
		}
		sliceGroups["global"] = globalSG
	}
	return nil
}
