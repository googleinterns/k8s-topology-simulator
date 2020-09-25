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
	"sort"

	"github.com/googleinterns/k8s-topology-simulator/modeling/types"
)

// endpointDeviation stores the deviation between expected number of endpoints
// and actual number of endpoints of a zone
type endpointDeviation struct {
	// zone name
	name string
	// deviation is the absolute value of the deviation = actual - expected
	deviation int
	// weights is used to indicate decimal endpoints deviation
	weight float64
	// consume by local zone in local-weighted-algorithm
	consumeByLocal bool
}

// endpointsList maintains the order of the endpoints available/needed, used as
// a list
type endpointsList struct {
	byZone []endpointDeviation
}

// push to back
func (el *endpointsList) push(newValue endpointDeviation) {
	el.byZone = append(el.byZone, newValue)
}

// push to front
func (el *endpointsList) pushFront(newValue endpointDeviation) {
	el.byZone = append([]endpointDeviation{newValue}, el.byZone...)
}

// pop the front
func (el *endpointsList) pop() {
	if len(el.byZone) == 0 {
		return
	}
	el.byZone = el.byZone[1:]
}

// ZonePriorityQueue sorts zone based on endpoints distribution ratio deviation
// compared to nodes ratio
type ZonePriorityQueue struct {
	SliceGroups map[string]types.EndpointSliceGroup
	Region      types.RegionInfo
	// ZoneNames with min/max ratio deviation should be placed first
	ZoneNames []string
	// ReceiveEndpoint indicates if the zone is going to receive endpoints or give
	// out endpoints
	ReceiveEndpoint bool
}

// Len is number of zones in the queue
func (pq ZonePriorityQueue) Len() int {
	return len(pq.ZoneNames)
}

// Less returns true if i should be ahead of j
func (pq ZonePriorityQueue) Less(i, j int) bool {
	if !pq.ReceiveEndpoint {
		return pq.less(i, j)
	}
	return pq.less(j, i)
}

// helper function to detect if i should be ahead of j based on giving one
// endpoint out.
func (pq ZonePriorityQueue) less(i, j int) bool {
	zoneA := pq.ZoneNames[i]
	zoneB := pq.ZoneNames[j]
	if pq.SliceGroups[zoneA].NumberOfEndpoints() == 0 {
		return false
	}
	if pq.SliceGroups[zoneB].NumberOfEndpoints() == 0 {
		return true
	}
	// If this queue is to receive endpoints, the zone with a higher traffic
	// load deviation should be placed first, deviation = expectedEndpoints /
	// actual endpoints = nodes ratio / actual endpoints
	if pq.ReceiveEndpoint {
		return pq.Region.ZoneDetails[zoneA].NodesRatio/float64(pq.SliceGroups[zoneA].NumberOfEndpoints()) <
			pq.Region.ZoneDetails[zoneB].NodesRatio/float64(pq.SliceGroups[zoneB].NumberOfEndpoints())
	}
	// If this queue is to give out endpoints, the zone with a lowe traffic load
	// after giving out one endpoint should be placed first
	return pq.Region.ZoneDetails[zoneA].NodesRatio/float64(pq.SliceGroups[zoneA].NumberOfEndpoints()-1) <
		pq.Region.ZoneDetails[zoneB].NodesRatio/float64(pq.SliceGroups[zoneB].NumberOfEndpoints()-1)
}

// Pop returns the first element in the queue and erases it
func (pq *ZonePriorityQueue) Pop() interface{} {
	n := len(pq.ZoneNames)
	zoneName := pq.ZoneNames[n-1]
	pq.ZoneNames = pq.ZoneNames[0 : n-1]
	return zoneName
}

// Push adds one element to the end of slice
func (pq *ZonePriorityQueue) Push(x interface{}) {
	pq.ZoneNames = append(pq.ZoneNames, x.(string))
}

// Swap swaps zone names at index i and j
func (pq *ZonePriorityQueue) Swap(i, j int) {
	pq.ZoneNames[i], pq.ZoneNames[j] = pq.ZoneNames[j], pq.ZoneNames[i]
}

// sortZoneByNames sorts the map by keys and returns an array of the sorted
// zoneNames. It helps traverse the map with a deterministic order
func sortZoneByNames(zones map[string]types.Zone) []string {
	var names []string
	for name := range zones {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// assignEndpoints helps distribute endpoints from rich zones to poor zones in
// local based algorithms
func assignEndpoints(receiveZone *endpointDeviation, endpointsAvailable *endpointsList, sliceGroups map[string]types.EndpointSliceGroup) {
	// traverse available zones to assign endpoints to receiving zone
	for index := 0; index < len(endpointsAvailable.byZone); {
		sendZone := endpointsAvailable.byZone[index]
		// if extra endpoints from available zones == required endpoints from
		// receiving zones, assign endpoints only
		if sendZone.deviation == receiveZone.deviation {
			sliceGroups[receiveZone.name].Composition[sendZone.name] = types.WeightedEndpoints{Number: sendZone.deviation, Weight: 1}
			receiveZone.deviation = 0
			endpointsAvailable.pop()
			break
		}
		// if extra endpoints from available zones > required endpoints from
		// receiving zones, assign endpoints and move to a new receiving zone
		if sendZone.deviation > receiveZone.deviation {
			sliceGroups[receiveZone.name].Composition[sendZone.name] = types.WeightedEndpoints{Number: receiveZone.deviation, Weight: 1}
			endpointsAvailable.byZone[index].deviation -= receiveZone.deviation
			receiveZone.deviation = 0
			break
		}
		// if extra endpoints from available zones < required endpoints from
		// receiving zones, assign endpoints and move to a new available zone
		if sendZone.deviation < receiveZone.deviation {
			sliceGroups[receiveZone.name].Composition[sendZone.name] = types.WeightedEndpoints{Number: sendZone.deviation, Weight: 1}
			receiveZone.deviation -= sendZone.deviation
			endpointsAvailable.pop()
		}
	}
}
