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

package simulator

import "github.com/googleinterns/k8s-topology-simulator/modeling/types"

// TrafficSimulator interface for different simulators
type TrafficSimulator interface {
	// Simulate simulates traffic with the provided regionInfo and
	// EndpointSliceGroups and returns a SimulationResult.
	Simulate(types.RegionInfo, map[string]types.EndpointSliceGroup) (types.SimulationResult, error)
}
