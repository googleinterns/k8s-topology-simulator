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
	"errors"
	"math"

	"github.com/googleinterns/k8s-topology-simulator/modeling/types"
)

// SharedGlobalAlgorithmCore takes multiple zones as input and output
// EntpointSliceGroups composition based on their nodes and endpoints:
// 1. EndpointSlices will be considered global by default
// 2. Once the number of endpoints in a zone reaches a specified threshold,
//	  zone-specific EndpointSlices will begin to be produced.
// 3. Kube Proxy will consume zone specific EndpointSlices. And decide whether
//    to consume the global EndpointSlice based on users input.
type SharedGlobalAlgorithmCore struct {
	// Weight of global EndpointSliceGroup
	globalWeight float64
	// Threshold of global EndpointSliceGroup that if the total number of endpoints
	// <= threshold, all endpoints go to global EndpointSliceGroup
	globalThreshold int
}

// CreateSliceGroups takes a region of zones as input and output
// EndpointSliceGroups
func (alg SharedGlobalAlgorithmCore) CreateSliceGroups(region types.RegionInfo, excludeContributor bool) (map[string]types.EndpointSliceGroup, error) {
	if region.ZoneDetails == nil {
		return nil, errors.New("can't create EndpointSlices without zones specified")
	}
	if region.TotalEndpoints <= alg.globalThreshold {
		return OriginalAlgorithm{}.CreateSliceGroups(region)
	}
	// The deviation for the traffic and capacity above
	deviation := make(map[string]float64)
	for _, zone := range region.ZoneDetails {
		// Calculate the deviation based on the capacity(endpoints) and
		// traffic(nodes) ratio
		deviation[zone.Name] = float64(zone.Endpoints) - float64(region.TotalEndpoints)*zone.NodesRatio
	}

	// Output EndpointSlices
	sliceGroups := make(map[string]types.EndpointSliceGroup)
	// globalSG is shared among all the zones
	var globalSliceGroup types.EndpointSliceGroup
	globalSliceGroup.Label = "global"
	globalSliceGroup.Composition = make(map[string]types.WeightedEndpoints)
	globalSliceGroup.ZoneTrafficWeights = make(map[string]float64)
	for name, zone := range region.ZoneDetails {
		var globalEndpoints types.WeightedEndpoints
		// calculate the global contribution of current zone based on the global
		// weight and the deviation of this zone If deviation > 0, this zone has
		// more endpoints compared to the ratio of nodes. It should contribute
		// the extra endpoints to the global sliceGroup with the weight counted.
		globalEndpoints.Number = int(math.Min(math.Max(0.0, deviation[name])/alg.globalWeight, float64(zone.Endpoints)))
		globalEndpoints.Weight = 1

		globalSliceGroup.Composition[name] = globalEndpoints
		globalSliceGroup.ZoneTrafficWeights[name] = alg.globalWeight
		if excludeContributor && globalEndpoints.Number != 0 && zone.Endpoints-globalEndpoints.Number != 0 {
			globalSliceGroup.ZoneTrafficWeights[name] = 0
		}

		// Calculate how many endpoints remain in the local zone
		var localGroup types.EndpointSliceGroup
		localGroup.Label = name
		localGroup.Composition = make(map[string]types.WeightedEndpoints)
		localGroup.ZoneTrafficWeights = make(map[string]float64)
		localGroup.Composition[name] = types.WeightedEndpoints{Number: zone.Endpoints - globalEndpoints.Number, Weight: 1}
		localGroup.ZoneTrafficWeights[name] = 1.0

		sliceGroups[name] = localGroup
	}
	sliceGroups[globalSliceGroup.Label] = globalSliceGroup
	return sliceGroups, nil
}
