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

// Zone abstracts the conception of 'zone' in clouds
type Zone struct {
	// Nodes is the numer of nodes of this zone
	Nodes int
	// Endpoints is the Number of endpoints in this zone
	Endpoints int
	// Name of this zone
	Name string
	// Number of pods / total number of pods of all zones
	endpointsRatio float64
	// Number of nodes / total number of nodes of all zones
	nodesRatio float64
}

// EndpointSliceGroup represents all the EndpointSlices under a same label, one
// group may be made up by many EndpointSlices (when the number of endpoints
// excceeds the capacity of one EndpointSlice). Since for now there is no need
// to know how the group is composed, we keep them as a whole
type EndpointSliceGroup struct {
	// Label is a unique identifier for an EndpointSliceGroup. This often
	// represents a topology label that the group will be consumed from.
	Label string
	// Composition stores contribution of endpoints in this group from different
	// zones
	Composition map[string]weightedEndpoints
	// ZoneTrafficWeights this sliceGroup has for requests from different zones
	ZoneTrafficWeights map[string]float64
}

// SimulationResult is to collect metrics of a simulation result
type SimulationResult struct {
	// InZoneTraffic is the total ratio of traffic that stays in the same zone
	InZoneTraffic float64
	// TrafficDistribution groups zoneTraffic by zone name
	TrafficDistribution map[string]zoneTraffic
	// MaxDeviation of traffic load of all endpoints
	MaxDeviation float64
	// MeanDeviation of traffic load of all endpoints
	MeanDeviation float64
	// DeviationSD represents the standard deviation of the daviation of traffic
	// load across all endpoints
	DeviationSD float64
}

type regionInfo struct {
	// total number of nodes of all zones
	totalNodes int
	// total number of endpoints of all zones
	totalEndpoints int
	// detailed information of each zone, zone name - exact zone
	zoneDetails map[string]Zone
}

// endpoints with weights are used to do routing inside an EndpointSliceGroup
type weightedEndpoints struct {
	// number of endpoints
	number int
	// weights of these endpoints when routing in a slice
	weight float64
}

// zoneTraffic records the detailed traffic infomation of a zone
type zoneTraffic struct {
	// zoneName of a specific zone
	zoneName string
	// incoming traffic this zone received
	incoming float64
	// outgoing traffic distribution of this zone
	outgoing map[string]float64
	// trafficLoad: ratio between exact traffic received by the zone and its
	// expected receiving traffic
	trafficLoad float64
	// zoneTrafficDetail stores detailed traffic load information for all
	// endpoints in the zone
	zoneTrafficDetail endpointsTraffic
}

// endpointsTraffic stores traffic load details of endpoints in a zone
type endpointsTraffic struct {
	// endpointsTrafficLoad for different endpoints belong to a zone in
	// different sliceGroups
	// key: sliceGroup label endpoints assigned to
	endpointsTrafficLoad map[string]float64
	// endpointsTrafficLoadDeviation for different endpoints belong to a zone
	// in different sliceGroups
	// key: sliceGroup label endpoints assigned to
	endpointsTrafficLoadDeviation map[string]float64
	// maxDeviationSG (SG:sliceGroup) of endpoints in a zone
	maxDeviationSG string
	// meanDeviation of endpoints in a zone
	meanDeviation float64
}

// Helper function to calculate number of endpoints of a specific
// EndpointSliceGroup
func (e EndpointSliceGroup) numberOfEndpoints() int {
	total := 0
	for _, endpoints := range e.Composition {
		total += endpoints.number
	}
	return total
}

// Helper function to calculate weighted number of endpoints of a specific
// EndpointSliceGroup
func (e EndpointSliceGroup) numberOfWeightedEndpoints() float64 {
	total := 0.0
	for _, endpoints := range e.Composition {
		total += float64(endpoints.number) * endpoints.weight
	}
	return total
}

// Helper function to create regionInfo with zone infos
func createRegionInfo(zones []Zone) (regionInfo, error) {
	if len(zones) == 0 {
		return regionInfo{}, errors.New("creating zoneinfos with zero length []Zone")
	}
	var totalEndpoints, totalNodes int

	region := regionInfo{zoneDetails: make(map[string]Zone)}
	for _, zone := range zones {
		if zone.Endpoints < 0 || zone.Nodes < 0 {
			return regionInfo{}, errors.New("invalid zones with number of nodes or endpoints < 0")
		}
		totalEndpoints += zone.Endpoints
		totalNodes += zone.Nodes
	}
	region.totalEndpoints = totalEndpoints
	region.totalNodes = totalNodes
	for _, zone := range zones {
		if totalEndpoints == 0 {
			zone.endpointsRatio = 0
		} else {
			zone.endpointsRatio = float64(zone.Endpoints) / float64(totalEndpoints)
		}
		if totalNodes == 0 {
			zone.nodesRatio = 0
		} else {
			zone.nodesRatio = float64(zone.Nodes) / float64(totalNodes)
		}
		region.zoneDetails[zone.Name] = zone
	}
	return region, nil
}
