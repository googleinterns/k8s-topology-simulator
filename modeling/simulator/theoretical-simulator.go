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

import (
	"errors"
	"math"

	"github.com/googleinterns/k8s-topology-simulator/modeling/types"
)

// TheoreticalSimulator calculates the theoretical probability of the traffic
// distribution
type TheoreticalSimulator struct{}

// Simulate calculates the theoretical distribution of the traffic
func (sim TheoreticalSimulator) Simulate(region types.RegionInfo, endpointSlices map[string]types.EndpointSliceGroup) (types.SimulationResult, error) {
	if len(region.ZoneDetails) == 0 || len(endpointSlices) == 0 {
		return types.SimulationResult{}, errors.New("can't evaluate probability based on empty zones or endpointslices")
	}

	zoneTrafficDetails := zoneSGDetails{}
	for zone := range region.ZoneDetails {
		zoneTrafficDetails[zone] = sliceGroupDetails{}
	}

	zoneTrafficDetails.getReachableEndpoints(endpointSlices)
	zoneTrafficDetails.getTraffic()
	zoneTrafficDetails.getEndpointsTrafficLoadDetails(region, endpointSlices)
	zoneTrafficToZone := zoneTrafficDetails.getZoneToZoneTraffic(region, endpointSlices)

	return getSimulationResult(zoneTrafficDetails, region, endpointSlices, zoneTrafficToZone), nil
}

// zoneSGDetails maps zone to its detailed traffic info
type zoneSGDetails map[string]sliceGroupDetails

type sliceGroupDetails struct {
	// number of endpoints reachable for a zone in each sliceGroup
	zoneReachableEndpoints map[string]float64
	// number of endpoints reachable for a zone in all sliceGroups
	zoneReachableEndpointsAll float64
	// traffic ratio of a zone to each sliceGroup
	zoneTrafficRatio map[string]float64
	// traffic load of endpoints belong to a zone in each sliceGroup
	endpointsTrafficLoad map[string]float64
	// traffic load deviation of endpoints belong to a zone in each sliceGroup
	endpointsTrafficLoadDeviation map[string]float64
}

// get reachable endpoints for every zone
func (zd zoneSGDetails) getReachableEndpoints(endpointSlices map[string]types.EndpointSliceGroup) {
	for zone, sgDetails := range zd {
		sgDetails.zoneReachableEndpoints = map[string]float64{}
		for sliceLabel, slice := range endpointSlices {
			sgDetails.zoneReachableEndpoints[sliceLabel] = float64(slice.NumberOfEndpoints()) * slice.ZoneTrafficWeights[zone]
			sgDetails.zoneReachableEndpointsAll += sgDetails.zoneReachableEndpoints[sliceLabel]
		}
		zd[zone] = sgDetails
	}
}

// get traffic distribution to sliceGroups for every zone
func (zd zoneSGDetails) getTraffic() {
	for zone, sgDetails := range zd {
		sgDetails.zoneTrafficRatio = map[string]float64{}
		if sgDetails.zoneReachableEndpointsAll == 0 {
			zd[zone] = sgDetails
			continue
		}
		for label := range sgDetails.zoneReachableEndpoints {
			sgDetails.zoneTrafficRatio[label] = sgDetails.zoneReachableEndpoints[label] / sgDetails.zoneReachableEndpointsAll
		}
		zd[zone] = sgDetails
	}
}

// get endpoints traffic load and its deviation in different sliceGroups
func (zd zoneSGDetails) getEndpointsTrafficLoadDetails(region types.RegionInfo, endpointSlices map[string]types.EndpointSliceGroup) {
	// total ratio of traffic received by each EndpointSliceGroup
	sgTrafficRatio := map[string]float64{}
	for zone, sgDetails := range zd {
		for label, trafficRatio := range sgDetails.zoneTrafficRatio {
			sgTrafficRatio[label] += region.ZoneDetails[zone].NodesRatio * trafficRatio
		}
	}

	// theoretically, traffic should be distributed equally among all the
	// endpoints
	theoreticalTrafficLoad := 1.0 / float64(region.TotalEndpoints)

	for zone, sgDetails := range zd {
		sgDetails.endpointsTrafficLoad = map[string]float64{}
		sgDetails.endpointsTrafficLoadDeviation = map[string]float64{}
		for label, sliceGroup := range endpointSlices {
			if sliceGroup.Composition[zone].Number == 0 || sliceGroup.NumberOfWeightedEndpoints() == 0 {
				continue
			}
			// calcualte the ratio of the endpoints in the sliceGroup
			zoneRatioInSG := float64(sliceGroup.Composition[zone].Number) * sliceGroup.Composition[zone].Weight / sliceGroup.NumberOfWeightedEndpoints()
			// zone endpoints traffic load in this sliceGroup = sliceGroup
			// traffic * zone ratio in this sliceGroup
			trafficLoad := sgTrafficRatio[label] * zoneRatioInSG / float64(sliceGroup.Composition[zone].Number)
			sgDetails.endpointsTrafficLoad[label] = trafficLoad
			sgDetails.endpointsTrafficLoadDeviation[label] = trafficLoad/theoreticalTrafficLoad - 1.0
		}
		zd[zone] = sgDetails
	}
}

// get traffic distribution between zones
func (zd zoneSGDetails) getZoneToZoneTraffic(region types.RegionInfo, endpointSlices map[string]types.EndpointSliceGroup) map[string]map[string]float64 {
	// ratio of traffic from a zone to other zones
	zoneTrafficToZone := map[string]map[string]float64{}
	for oriZone := range region.ZoneDetails {
		zoneTrafficToZone[oriZone] = map[string]float64{}
		for label, sliceGroup := range endpointSlices {
			if sliceGroup.NumberOfWeightedEndpoints() == 0 {
				continue
			}
			for destZone := range region.ZoneDetails {
				desZoneRatioInSG := float64(sliceGroup.Composition[destZone].Number) * sliceGroup.Composition[destZone].Weight / sliceGroup.NumberOfWeightedEndpoints()
				// traffic oriZone -> desZone: sum(traffic distribution of
				// oriZone * traffic ratio from oriZone to this sliceGroup *
				// desZone ratio in this sliceGroup)
				zoneTrafficToZone[oriZone][destZone] += region.ZoneDetails[oriZone].NodesRatio * zd[oriZone].zoneTrafficRatio[label] * desZoneRatioInSG
			}
		}
	}
	return zoneTrafficToZone
}

// calculate simulation result based on probabilities
func getSimulationResult(zd zoneSGDetails, region types.RegionInfo, endpointSlices map[string]types.EndpointSliceGroup, zoneTrafficToZone map[string]map[string]float64) types.SimulationResult {

	// calculate result of one simulation
	var simResult types.SimulationResult
	// traffic distribution details by zone
	simResult.TrafficDistribution = map[string]types.ZoneTraffic{}

	var totalDeviation float64
	var maxDeviation float64
	for zoneName, zoneInfo := range region.ZoneDetails {
		// zoneX -> zoneX forms inzone traffic
		simResult.InZoneTraffic += zoneTrafficToZone[zoneName][zoneName]
		zoneMaxDeviation := 0.0
		zoneDeviation := 0.0
		var maxLabel string
		for label, deviation := range zd[zoneName].endpointsTrafficLoadDeviation {
			zoneDeviation += math.Abs(deviation) * float64(endpointSlices[label].Composition[zoneName].Number)
			if math.Abs(deviation) > zoneMaxDeviation {
				zoneMaxDeviation = math.Abs(deviation)
				maxLabel = label
			}
		}
		totalDeviation += zoneDeviation
		maxDeviation = math.Max(zoneMaxDeviation, maxDeviation)

		var traffic types.ZoneTraffic
		traffic.ZoneName = zoneName
		// Outgoing traffic distribution
		traffic.Outgoing = zoneTrafficToZone[zoneName]
		for originZoneName, originZone := range region.ZoneDetails {
			// Accumulate total incoming traffic to zoneName
			traffic.Incoming += originZone.NodesRatio * zoneTrafficToZone[originZoneName][zoneName]
		}
		traffic.TrafficLoad = traffic.Incoming / zoneInfo.EndpointsRatio
		traffic.ZoneTrafficDetail.MaxDeviationSG = maxLabel
		traffic.ZoneTrafficDetail.MeanDeviation = zoneDeviation / float64(zoneInfo.Endpoints)
		traffic.ZoneTrafficDetail.EndpointsTrafficLoad = zd[zoneName].endpointsTrafficLoad
		traffic.ZoneTrafficDetail.EndpointsTrafficLoadDeviation = zd[zoneName].endpointsTrafficLoadDeviation

		simResult.TrafficDistribution[zoneName] = traffic
	}

	meanDeviation := totalDeviation / float64(region.TotalEndpoints)
	var squareSum float64
	var deviationSD float64
	// calculate standard deviation of traffic load deviation
	for zone := range zd {
		for label, deviation := range zd[zone].endpointsTrafficLoadDeviation {
			squareSum += math.Pow(deviation-meanDeviation, 2) * float64(endpointSlices[label].Composition[zone].Number)
		}
	}
	deviationSD = math.Sqrt(squareSum / float64(region.TotalEndpoints))

	simResult.MaxDeviation = maxDeviation
	simResult.MeanDeviation = meanDeviation
	simResult.DeviationSD = deviationSD
	return simResult
}
