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

// SharedGlobalAlgorithm takes multiple zones as input and output endpointslices
// composition based on their nodes and endpoints:
// 1. Endpointslices will be considered global by default
// 2. Once the number of endpoints in a zone reaches a specified threshold,
//	  zone-specific EndpointSlices will begin to be produced.
// 3. Kube Proxy will consume both global and zone specific EndpointSlices.
//	  Endpoints in the same zone will be given a higher weight.
type SharedGlobalAlgorithm struct {
	// Weight of global endpointslices
	globalWeight float64
	// Threshold of global endpointslices that if the total number of endpoints
	// <= threshold, all endpoints go to global endpointslice
	//	 Int should be enough
	globalThreshold int
}

// CreateSlices takes multiple zones as input and output endpointslices
// globalweight and globalthreshold set before are two paramters for the global
// endpointslice
func (alg SharedGlobalAlgorithm) CreateSlices(zones zoneInfos) (map[string]EndpointSliceGroup, error) {
	if zones.zoneDetails == nil {
		return nil, errors.New("Can't create endpointslices with 0 number of zone")
	}
	// The deviation for the traffic and capacity above
	deviation := make(map[string]float64)

	for _, zone := range zones.zoneDetails {
		// Calculate the deviation based on the capacity(endpoints) and
		// traffic(nodes) ratio
		deviation[zone.Name] = float64(zone.Endpoints) - float64(zones.totalEndpoints)*zone.nodesRatio
	}

	// Output endpointslices
	endpointslices := make(map[string]EndpointSliceGroup)
	// The 'big' global slice -- might be split into many small global slices
	// when the number of endpoints > required number of endpoints/endpoinslice,
	// i.e. 100 for default not able to divide the big one into smaller ones that
	// the contributions may vary and there is no need to do so either.
	var globalSlice EndpointSliceGroup
	globalSlice.Label = "global"
	globalSlice.Composition = make(map[string]int)
	globalSlice.Weights = make(map[string]float64)
	for name, zone := range zones.zoneDetails {
		var globalEndpoints int
		// If total pods <= threshold, all pods go to global slice
		if zones.totalEndpoints <= alg.globalThreshold {
			globalEndpoints = zone.Endpoints
		} else {
			// Otherwise calculate the global contribution of current zone based
			// on the global weight and the deviation of this zone
			globalEndpoints = int(math.Min(math.Max(0.0, deviation[name])/alg.globalWeight, float64(zone.Endpoints)))
		}

		globalSlice.Composition[name] = globalEndpoints
		globalSlice.Weights[name] = alg.globalWeight

		// Calculate how many endpoints remain in the local zone
		var slice EndpointSliceGroup
		slice.Label = name
		slice.Composition = make(map[string]int)
		slice.Weights = make(map[string]float64)
		slice.Composition[name] = zone.Endpoints - globalEndpoints
		slice.Weights[name] = 1.0

		endpointslices[name] = slice
	}
	endpointslices[globalSlice.Label] = globalSlice
	return endpointslices, nil
}

// CreateAlg -- the constructor of the algorithm , set weight and threshold of
// the global endpointslice
func CreateAlg(weight float64, threshold int) (SharedGlobalAlgorithm, error) {
	if weight < 0 || threshold < 0 {
		return SharedGlobalAlgorithm{}, errors.New("Invalid weight/threshold values to init algorihtm")
	}
	alg := SharedGlobalAlgorithm{globalWeight: weight, globalThreshold: threshold}
	return alg, nil
}
