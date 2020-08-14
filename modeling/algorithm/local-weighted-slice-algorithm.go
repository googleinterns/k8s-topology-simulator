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

// LocalWeightedSliceAlgorithm is a variation of LocalSliceAlgorithm which
// 'borrows' and 'rents' endpoints from other zones to make the local
// EndpointSlice balanced with the incoming traffic. This variation uses weights
// to make precise distribution without float to int approximation
type LocalWeightedSliceAlgorithm struct{}

// CreateSliceGroups creates sliceGroups with weights to indicate float
// endpoints. Zones will have local sliceGroup representing integer number of
// endpoints while having a shared sliceGroup with weights representing float
// number of endpoints
func (alg LocalWeightedSliceAlgorithm) CreateSliceGroups(region types.RegionInfo) (map[string]types.EndpointSliceGroup, error) {
	if region.ZoneDetails == nil {
		return nil, fmt.Errorf("zoneDetail should not be nil")
	}
	sliceGroups := map[string]types.EndpointSliceGroup{}
	// endpointsAvailable stores zones with int number of endpoints available
	endpointsAvailable := endpointsList{}
	// endpointsNeeded stores zones with int number of endpoints needed
	endpointsNeeded := endpointsList{}
	// weightedEndpointsAvailable stores zones with float number of endpoints
	// available
	weightedEndpointsAvailable := endpointsList{}
	// weightedEndpointsNeeded stores zones with float number of endpoints
	// needed
	weightedEndpointsNeeded := endpointsList{}
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
		// deviation: a negative value means this zone needs more endpoints from
		// other zones, a positive value means this zone needs to give out
		// endpoints to other zones
		deviation := float64(zone.Endpoints) - expectedEndpoints
		// intDeviation is dealt same way as original local-slice algorithm that
		// directly 'borrows' or 'sends' endpoints among zones
		intDeviation := int(deviation)
		if intDeviation == 0 {
			localGroup.Composition[zoneName] = types.WeightedEndpoints{Number: int(expectedEndpoints), Weight: 1}
		} else if intDeviation > 0 {
			endpointsAvailable.push(endpointDeviation{name: zoneName, deviation: intDeviation})
			localGroup.Composition[zoneName] = types.WeightedEndpoints{Number: int(expectedEndpoints), Weight: 1}
		} else {
			endpointsNeeded.push(endpointDeviation{name: zoneName, deviation: -intDeviation})
			localGroup.Composition[zoneName] = types.WeightedEndpoints{Number: zone.Endpoints, Weight: 1}
		}
		sliceGroups[zoneName] = localGroup

		// push 0.xx deviation into corresponding lists
		// 2.3 expected endpoints, 3 ownd endpoints, 0.7 decimal deviation, 0.3
		// endpoints and 2 endpoints needed
		// 2.4 expected endpoints, 1 owned endpoint, -0.4 decimal deviation, 0.4
		// endpoints and 2 endpoints needed
		decimalDeviation := deviation - float64(intDeviation)
		if decimalDeviation > 0 {
			// One endpoint from this zone will contribute 1-decimalDeviation to
			// the local zone, and contribute remaining to other zones. This
			// will be implemented through routing weights of EndpointSliceGroup
			weightedEndpointsAvailable.push(endpointDeviation{name: zoneName, deviation: 1, weight: 1 - decimalDeviation, consumeByLocal: true})
		} else if decimalDeviation < 0 {
			weightedEndpointsNeeded.push(endpointDeviation{name: zoneName, deviation: 1, weight: -decimalDeviation})
		}
	}

	err := alg.balanceSliceGroups(&endpointsAvailable, &endpointsNeeded, &weightedEndpointsAvailable, &weightedEndpointsNeeded, sliceGroups)
	return sliceGroups, err
}

// balanceSliceGroups distributes endpoints from zones with extra endpoints to
// EndpointSliceGroups for zones with insufficient endpoints.
func (alg LocalWeightedSliceAlgorithm) balanceSliceGroups(endpointsAvailable *endpointsList, endpointsNeeded *endpointsList, weightedEndpointsAvailable *endpointsList, weightedEndpointsNeeded *endpointsList, sliceGroups map[string]types.EndpointSliceGroup) error {

	for _, receiveZone := range endpointsNeeded.byZone {
		// There are no more full endpoints available, but this zone still needs
		// endpoints. Push the needed endpoints into a weighted list and deal
		// with them as partial endpoints.
		if len(endpointsAvailable.byZone) == 0 {
			receiveZone.weight = 1
			weightedEndpointsNeeded.push(receiveZone)
			endpointsNeeded.pop()
			continue
		}
		// same logic as original local-slice algorithm
		for index := 0; index < len(endpointsAvailable.byZone); {
			sendZone := endpointsAvailable.byZone[index]
			if sendZone.deviation == receiveZone.deviation {
				sliceGroups[receiveZone.name].Composition[sendZone.name] = types.WeightedEndpoints{Number: sendZone.deviation, Weight: 1}
				receiveZone.deviation = 0
				endpointsAvailable.pop()
				break
			}
			if sendZone.deviation > receiveZone.deviation {
				sliceGroups[receiveZone.name].Composition[sendZone.name] = types.WeightedEndpoints{Number: receiveZone.deviation, Weight: 1}
				endpointsAvailable.byZone[index].deviation -= receiveZone.deviation
				receiveZone.deviation = 0
				break
			}
			if sendZone.deviation < receiveZone.deviation {
				sliceGroups[receiveZone.name].Composition[sendZone.name] = types.WeightedEndpoints{Number: sendZone.deviation, Weight: 1}
				receiveZone.deviation -= sendZone.deviation
				endpointsAvailable.pop()
				continue
			}
		}
		// if needed.deviation != 0 means more full endpoints needed than
		// available, push to weighted list and deal them as partial endpoints
		if receiveZone.deviation != 0 {
			receiveZone.weight = 1
			weightedEndpointsNeeded.push(receiveZone)
		}
		endpointsNeeded.pop()
	}
	// If 'int' endpoints available more than 'int' endpoints needed, push the
	// extra endpoints into weighted list and deal as float endpoints, this
	// distinguishes from the case where endpoints are partially consumed by its
	// original zone as described above.
	for _, extraEndpoints := range endpointsAvailable.byZone {
		extraEndpoints.weight = 1
		weightedEndpointsAvailable.push(extraEndpoints)
		endpointsAvailable.pop()
	}

	// distribute 'float' endpoints with weights. create shared slices among
	// zones that ends up with sum(weights for each zone in the SG) = 1
	// use weights to implement float endpoints for a zone, i.e. 0.4 endpoints
	// for zoneA will be implemented with a SG having 1 endpoint but 0.4 routing
	// weight to zoneA
	for _, extraEndpoints := range weightedEndpointsAvailable.byZone {
		sharedSlice := types.EndpointSliceGroup{Label: "shared", Composition: map[string]types.WeightedEndpoints{}, ZoneTrafficWeights: map[string]float64{}}
		// If this endpoint will be consumbed by its local zone, contribue to
		// the local zone first then use the remaining 'weights' to share with
		// other zones
		if extraEndpoints.consumeByLocal {
			// In this case, the weight of extraEndpoints is the weight needed
			// by its local zone. After contributing to the local zone, use the
			// remaining weights to serve other zones.
			sharedSlice.ZoneTrafficWeights[extraEndpoints.name] = extraEndpoints.weight
			sharedSlice.Label += "-" + extraEndpoints.name
			extraEndpoints.weight = 1 - extraEndpoints.weight
			extraEndpoints.consumeByLocal = false
		}
		weightedEndpoint := types.WeightedEndpoints{Number: extraEndpoints.deviation, Weight: 1}
		sharedSlice.Composition[extraEndpoints.name] = weightedEndpoint
		// similar logic as 'int' endpoints distribution
		for index := 0; index < len(weightedEndpointsNeeded.byZone); {
			receiveZone := weightedEndpointsNeeded.byZone[index]
			// float endpoints = number * weight
			deviation := float64(receiveZone.deviation)*receiveZone.weight - float64(extraEndpoints.deviation)*extraEndpoints.weight
			if deviation == 0 {
				sharedSlice.ZoneTrafficWeights[receiveZone.name] += extraEndpoints.weight
				sharedSlice.Label += "-" + receiveZone.name
				weightedEndpointsNeeded.pop()
				break
			}
			// needed > available, update needed value and consume the next
			// available one
			if deviation > 0 {
				sharedSlice.ZoneTrafficWeights[receiveZone.name] += extraEndpoints.weight
				sharedSlice.Label += "-" + receiveZone.name
				weightedEndpointsNeeded.byZone[index].deviation = 1
				weightedEndpointsNeeded.byZone[index].weight = deviation
				break
			}
			// needed < available, update the available value and serve the next
			// needed one
			if deviation < 0 {
				sharedSlice.ZoneTrafficWeights[receiveZone.name] += float64(receiveZone.deviation) * receiveZone.weight / float64(extraEndpoints.deviation)
				sharedSlice.Label += "-" + receiveZone.name
				extraEndpoints.weight -= sharedSlice.ZoneTrafficWeights[receiveZone.name]
				weightedEndpointsNeeded.pop()
				continue
			}
		}
		sliceGroups[sharedSlice.Label] = sharedSlice
		weightedEndpointsAvailable.pop()
	}
	return nil
}
