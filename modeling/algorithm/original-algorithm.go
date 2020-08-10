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

// OriginalAlgorithm equally distributes the traffic to all endpoints behind a
// servie, serving as a benchmark to evaluate other algorithms
type OriginalAlgorithm struct{}

// CreateSliceGroups puts all endpoints into a global EndpointSliceGroup
func (alg OriginalAlgorithm) CreateSliceGroups(region types.RegionInfo) (map[string]types.EndpointSliceGroup, error) {
	if region.ZoneDetails == nil {
		return nil, fmt.Errorf("zoneDetail should not be nil")
	}
	globalSG := types.EndpointSliceGroup{Label: "global",
		Composition:        map[string]types.WeightedEndpoints{},
		ZoneTrafficWeights: map[string]float64{},
	}
	for zoneName, zone := range region.ZoneDetails {
		globalSG.ZoneTrafficWeights[zoneName] = 1.0
		globalSG.Composition[zoneName] = types.WeightedEndpoints{Number: zone.Endpoints, Weight: 1.0}
	}
	return map[string]types.EndpointSliceGroup{"global": globalSG}, nil
}
