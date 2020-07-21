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

package modeling

import (
	"errors"
	"math"
)

// SharedGlobalAlgorithm takes multiple zones as input and output
// EntpointSliceGroups composition based on their nodes and endpoints:
// 1. EndpointSlices will be considered global by default
// 2. Once the number of endpoints in a zone reaches a specified threshold,
//	  zone-specific EndpointSlices will begin to be produced.
// 3. Kube Proxy will consume both global and zone specific EndpointSlices.
//	  Endpoints in the same zone will be given a higher weight.
type SharedGlobalAlgorithm struct {
	// Weight of global EndpointSliceGroup
	globalWeight float64
	// Threshold of global EndpointSliceGroup that if the total number of endpoints
	// <= threshold, all endpoints go to global EndpointSliceGroup
	globalThreshold int
}

// CreateSliceGroups takes a region of zones as input and output
// EndpointSliceGroups
func (alg SharedGlobalAlgorithm) CreateSliceGroups(region regionInfo) (map[string]EndpointSliceGroup, error) {
	if region.zoneDetails == nil {
		return nil, errors.New("can't create EndpointSlices with 0 number of zone")
	}
	// The deviation for the traffic and capacity above
	deviation := make(map[string]float64)

	for _, zone := range region.zoneDetails {
		// Calculate the deviation based on the capacity(endpoints) and
		// traffic(nodes) ratio
		deviation[zone.Name] = float64(zone.Endpoints) - float64(region.totalEndpoints)*zone.nodesRatio
	}

	// Output EndpointSlices
	sliceGroups := make(map[string]EndpointSliceGroup)
	// The global sliceGroup -- might be split into many small global slices
	// when the number of endpoints > required number of endpoints per
	// EndpointSlice, i.e. 100 for default. Not be able to divide the big one
	// into smaller ones that the contributions may vary and there is no need to
	// do so either.
	var globalSliceGroup EndpointSliceGroup
	globalSliceGroup.Label = "global"
	globalSliceGroup.Composition = make(map[string]int)
	globalSliceGroup.Weights = make(map[string]float64)
	for name, zone := range region.zoneDetails {
		var globalEndpoints int
		// If total pods <= threshold, all pods go to global slice
		if region.totalEndpoints <= alg.globalThreshold {
			globalEndpoints = zone.Endpoints
		} else {
			// Otherwise calculate the global contribution of current zone based
			// on the global weight and the deviation of this zone
			globalEndpoints = int(math.Min(math.Max(0.0, deviation[name])/alg.globalWeight, float64(zone.Endpoints)))
		}

		globalSliceGroup.Composition[name] = globalEndpoints
		globalSliceGroup.Weights[name] = alg.globalWeight

		// Calculate how many endpoints remain in the local zone
		var localGroup EndpointSliceGroup
		localGroup.Label = name
		localGroup.Composition = make(map[string]int)
		localGroup.Weights = make(map[string]float64)
		localGroup.Composition[name] = zone.Endpoints - globalEndpoints
		localGroup.Weights[name] = 1.0

		sliceGroups[name] = localGroup
	}
	sliceGroups[globalSliceGroup.Label] = globalSliceGroup
	return sliceGroups, nil
}

// CreateAlg -- the constructor of the algorithm , set weight and threshold of
// the global EndpointSliceGroup
func CreateAlg(weight float64, threshold int) (SharedGlobalAlgorithm, error) {
	if weight < 0 || threshold < 0 {
		return SharedGlobalAlgorithm{}, errors.New("invalid weight/threshold values to init algorithm")
	}
	alg := SharedGlobalAlgorithm{globalWeight: weight, globalThreshold: threshold}
	return alg, nil
}
