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

type regionInfo struct {
	// Total number of nodes of all zones
	totalNodes int
	// Total number of endpoints of all zones
	totalEndpoints int
	// Detailed information of each zone, zone name - exact zone
	zoneDetails map[string]Zone
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
	Composition map[string]int
	// Weights of endpoints in this group for different zones while routing
	Weights map[string]float64
}

// Traffic represents the detailed traffic infomation of a zone
type Traffic struct {
	// ZoneName of a specific zone
	ZoneName string
	// Incoming traffic that the same zone received
	Incoming float64
	// Outgoing traffic distribution of the same zone
	Outgoing map[string]float64
}

// SimulationResult is to collect metrics of a simulation result
type SimulationResult struct {
	// InZoneTraffic is the total ratio of traffic that stays in the same zone
	InZoneTraffic float64
	// TrafficDetail groups Traffic by zone name
	TrafficDetail map[string]Traffic
	// Workload balance for different zones -- ratio of incoming traffic / ratio
	// of capacity
	Workload map[string]float64
}

// Helper function to calculate number of endpoints of a specific
// EndpointSliceGroup
func (e EndpointSliceGroup) numberOfPods() int {
	total := 0
	for _, pods := range e.Composition {
		total += pods
	}
	return total
}

func createRegionInfo(zones []Zone) (regionInfo, error) {
	if len(zones) == 0 {
		return regionInfo{}, errors.New("creating zoneinfos with zero length []Zone")
	}
	var totalPods, totalNodes int

	region := regionInfo{zoneDetails: make(map[string]Zone)}
	for _, zone := range zones {
		if zone.Endpoints <= 0 || zone.Nodes <= 0 {
			return regionInfo{}, errors.New("invalid zones with number of nodes or endpoints <= 0")
		}
		totalPods += zone.Endpoints
		totalNodes += zone.Nodes
	}
	region.totalEndpoints = totalPods
	region.totalNodes = totalNodes
	for _, zone := range zones {
		zone.endpointsRatio = float64(zone.Endpoints) / float64(totalPods)
		zone.nodesRatio = float64(zone.Nodes) / float64(totalNodes)
		region.zoneDetails[zone.Name] = zone
	}
	return region, nil
}
