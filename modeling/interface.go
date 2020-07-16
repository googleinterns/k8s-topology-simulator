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

// RoutingAlgorithm interface for different routing algorithms
type RoutingAlgorithm interface {
	//This interface is to create endpointslices based on the current zones and
	//the rouing algorithm
	//	Input: zones that involved in the routing
	//	Output: endpointslices that created based on the routing rules
	CreateSlices(zoneInfos) (map[string]EndpointSliceGroup, error)
}

// TrafficSimulator interface for different simulators
type TrafficSimulator interface {
	//This interface is to simulate the traffic among the zones
	//	Input: zones and endpointslices
	//	Output: Simulation results
	Simulate(zoneInfos, map[string]EndpointSliceGroup) (Stat, error)
}
