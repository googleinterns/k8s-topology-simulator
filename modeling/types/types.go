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

package types

import "errors"

// Zone abstracts the conception of 'zone' in clouds
type Zone struct {
	// Nodes is the numer of nodes of this zone
	Nodes int
	// Endpoints is the Number of endpoints in this zone
	Endpoints int
	// Name of this zone
	Name string
	// EndpointsRatio of this zone compared to all endpoints
	EndpointsRatio float64
	// NodesRatio of this zone compared to all nodes
	NodesRatio float64
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
	Composition map[string]WeightedEndpoints
	// ZoneTrafficWeights this sliceGroup has for requests from different zones
	ZoneTrafficWeights map[string]float64
}

// SimulationResult is to collect metrics of a simulation result
type SimulationResult struct {
	// Invalid if something ends up with unexpected errors, i.e. some zones have
	// no endpoints to send traffic to, illegal routing weights (weights < 0)
	// etc.
	Invalid bool
	// InZoneTraffic is the total ratio of traffic that stays in the same zone
	InZoneTraffic float64
	// TrafficDistribution groups zoneTraffic by zone name
	TrafficDistribution map[string]ZoneTraffic
	// MaxDeviation of traffic load of all endpoints
	MaxDeviation float64
	// MeanDeviation of traffic load of all endpoints
	MeanDeviation float64
	// DeviationSD represents the standard deviation of the daviation of traffic
	// load across all endpoints
	DeviationSD float64
}

// RegionInfo wraps information of zones in a region
type RegionInfo struct {
	// TotalNodes of all zones
	TotalNodes int
	// TotalEndpoints of all zones
	TotalEndpoints int
	// ZoneDetails by zone
	ZoneDetails map[string]Zone
}

// WeightedEndpoints are used to do routing inside an EndpointSliceGroup
type WeightedEndpoints struct {
	// Number of endpoints
	Number int
	// Weight of these endpoints when routing in a slice
	Weight float64
}

// ZoneTraffic records the detailed traffic infomation of a zone
type ZoneTraffic struct {
	// ZoneName of a specific zone
	ZoneName string
	// Incoming traffic this zone received
	Incoming float64
	// Outgoing traffic distribution of this zone
	Outgoing map[string]float64
	// TrafficLoad: ratio between exact traffic received by the zone and its
	// expected receiving traffic
	TrafficLoad float64
	// ZoneTrafficDetail stores detailed traffic load information for all
	// endpoints in the zone
	ZoneTrafficDetail EndpointsTraffic
}

// EndpointsTraffic stores traffic load details of endpoints in a zone
type EndpointsTraffic struct {
	// EndpointsTrafficLoad for different endpoints belong to a zone in
	// different sliceGroups
	// key: sliceGroup label endpoints assigned to
	EndpointsTrafficLoad map[string]float64
	// EndpointsTrafficLoadDeviation for different endpoints belong to a zone
	// in different sliceGroups
	// key: sliceGroup label endpoints assigned to
	EndpointsTrafficLoadDeviation map[string]float64
	// MaxDeviationSG (SG:sliceGroup) of endpoints in a zone
	MaxDeviationSG string
	// MeanDeviation of endpoints in a zone
	MeanDeviation float64
}

// NumberOfEndpoints calculates number of endpoints of a specific
// EndpointSliceGroup
func (e EndpointSliceGroup) NumberOfEndpoints() int {
	total := 0
	for _, endpoints := range e.Composition {
		total += endpoints.Number
	}
	return total
}

// NumberOfWeightedEndpoints calculates weighted number of endpoints of a
// specific EndpointSliceGroup
func (e EndpointSliceGroup) NumberOfWeightedEndpoints() float64 {
	total := 0.0
	for _, endpoints := range e.Composition {
		total += float64(endpoints.Number) * endpoints.Weight
	}
	return total
}

// CreateRegionInfo creates regionInfo with zone infos
func CreateRegionInfo(zones []Zone) (RegionInfo, error) {
	if len(zones) == 0 {
		return RegionInfo{}, errors.New("creating zoneinfos with zero length []Zone")
	}
	var totalEndpoints, totalNodes int

	region := RegionInfo{ZoneDetails: make(map[string]Zone)}
	for _, zone := range zones {
		if zone.Endpoints < 0 || zone.Nodes < 0 {
			return RegionInfo{}, errors.New("invalid zones with number of nodes or endpoints < 0")
		}
		totalEndpoints += zone.Endpoints
		totalNodes += zone.Nodes
	}
	region.TotalEndpoints = totalEndpoints
	region.TotalNodes = totalNodes
	for _, zone := range zones {
		if totalEndpoints == 0 {
			zone.EndpointsRatio = 0
		} else {
			zone.EndpointsRatio = float64(zone.Endpoints) / float64(totalEndpoints)
		}
		if totalNodes == 0 {
			zone.NodesRatio = 0
		} else {
			zone.NodesRatio = float64(zone.Nodes) / float64(totalNodes)
		}
		region.ZoneDetails[zone.Name] = zone
	}
	return region, nil
}
