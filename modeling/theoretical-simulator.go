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

import "errors"

// TheoreticalSimulator calculates the theoretical probability of the traffic
// distribution
type TheoreticalSimulator struct{}

// Simulate calculates the theoretical distribution of the traffic
func (sim TheoreticalSimulator) Simulate(region regionInfo, endpointSlices map[string]EndpointSliceGroup) (SimulationResult, error) {
	if len(region.zoneDetails) == 0 || len(endpointSlices) == 0 {
		return SimulationResult{}, errors.New("can't evaluate probability based on empty zones or endpointslices")
	}
	// kube-proxy calculation, zones name - endpoints in potential destination sclies
	kubeProxy := make(map[string]map[string]float64)
	for name := range region.zoneDetails {
		kubeProxy[name] = map[string]float64{}
		for _, slice := range endpointSlices {
			for compZone, pods := range slice.Composition {
				kubeProxy[name][compZone] += float64(pods) * slice.Weights[name]
			}
		}
	}
	// probability of traffic from a zone to other zones, outgoing zone --
	// probability of going to other zones
	zoneEndpoints := map[string]map[string]float64{}
	for zoneName, endpointsDistribution := range kubeProxy {
		zoneTotal := 0.0
		for _, endpoints := range endpointsDistribution {
			zoneTotal += endpoints
		}
		zoneEndpoints[zoneName] = map[string]float64{}
		for destZoneName, endpoints := range endpointsDistribution {
			zoneEndpoints[zoneName][destZoneName] = endpoints / zoneTotal
		}
	}

	// calculate result of one simulation
	var simResult SimulationResult
	simResult.TrafficDetail = map[string]Traffic{}
	simResult.Workload = map[string]float64{}
	for zoneName, zone := range region.zoneDetails {
		// zoneX -> zoneX forms inzone traffic
		simResult.InZoneTraffic += zone.nodesRatio * zoneEndpoints[zoneName][zoneName]

		var traffic Traffic
		traffic.ZoneName = zoneName
		// Outgoing traffic distribution
		traffic.Outgoing = zoneEndpoints[zoneName]
		for originZoneName, originZone := range region.zoneDetails {
			// Accumulate total incoming traffic to zoneName
			traffic.Incoming += originZone.nodesRatio * zoneEndpoints[originZoneName][zoneName]
		}
		simResult.Workload[zoneName] = traffic.Incoming / zone.endpointsRatio
		simResult.TrafficDetail[zoneName] = traffic
	}
	return simResult, nil
}
