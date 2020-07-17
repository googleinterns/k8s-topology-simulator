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
	"fmt"
)

// Model wrapper class for the simulation components
type Model struct {
	slices    map[string]EndpointSliceGroup
	alg       RoutingAlgorithm
	simulator TrafficSimulator

	Zones   zoneInfos
	Results []Stat
	// number of max endpoints per slice
	SliceCapacity uint
}

// NewModel creates a model with zones, routing algorithm and traffic simulator
// and uses the algorithm to create the endpointslicegroups
func NewModel(zones []Zone, alg RoutingAlgorithm, sim TrafficSimulator) (*Model, error) {
	zoneInfo, err := createZoneinfos(zones)
	if err != nil {
		return nil, err
	}
	if alg == nil || sim == nil {
		return nil, errors.New("Can't create model with nil algorithm or simulator")
	}
	model := &Model{
		Zones:         zoneInfo,
		SliceCapacity: 100,
		alg:           alg,
		simulator:     sim,
	}
	slices, err := model.alg.CreateSlices(model.Zones)
	if err != nil {
		return nil, err
	}
	model.slices = slices
	return model, nil
}

// StartSimulation based on the zones and endpointslicegroups
func (m *Model) StartSimulation() error {
	stat, err := m.simulator.Simulate(m.Zones, m.slices)
	if err != nil {
		return err
	}
	m.Results = append(m.Results, stat)
	return nil
}

// UpdateAlgorithm updates the algorithm and creates the new
// endpointslicegroups based on the new algorithm
// TODO: should we erase the previous results?
func (m *Model) UpdateAlgorithm(alg RoutingAlgorithm) error {
	if alg == nil {
		return errors.New("Empty algorithm")
	}
	m.alg = alg
	slices, err := m.alg.CreateSlices(m.Zones)
	if err != nil {
		return err
	}
	m.slices = slices
	return nil
}

// PrintLastResult prints the summary of the last simulation result
func (m *Model) PrintLastResult() error {
	_, numberOfSlices, err := m.GetEndpointslices()
	if err != nil {
		return err
	}
	lastResult := m.Results[len(m.Results)-1]
	fmt.Printf("%% in-zone traffic \t %.2f%%\n", lastResult.InZoneTraffic*100)
	fmt.Printf("# of endpoint slices\t %v\n", numberOfSlices)
	fmt.Printf("# of endpoints\t %d\n", m.Zones.totalEndpoints)
	fmt.Println("----------------------------------------------")

	for zone, traffic := range lastResult.TrafficDetail {
		fmt.Printf("Total to %s \t %.f%% \n", zone, traffic.IncomingTraffic*100)
	}
	fmt.Println("----------------------------------------------")

	for zone, traffic := range lastResult.TrafficDetail {
		fmt.Printf("From %s : \n", zone)
		for z, t := range traffic.OutgoingTraffic {
			fmt.Printf("\t to %s : %.2f%% \n", z, t*100)
		}
	}
	fmt.Println("----------------------------------------------")

	for zone, workload := range lastResult.Workload {
		fmt.Printf("Workload for %s : \t %.2f%% \n", zone, workload*100)
	}
	return nil
}

// GetEndpointslices gets the created endpoitnslicegroups, number of
// endpointslices
func (m *Model) GetEndpointslices() (map[string]EndpointSliceGroup, uint, error) {
	if len(m.slices) == 0 {
		// To avoid the situation m.slices is not nil but is empty, intentionally
		// return the nil for the first value
		return nil, 0, nil
	}
	totalSlice := uint(0)
	for _, slice := range m.slices {
		pods := slice.numberOfPods()
		totalSlice += uint(pods) / m.SliceCapacity
		if uint(pods)%m.SliceCapacity != 0 {
			totalSlice++
		}
	}
	return m.slices, totalSlice, nil
}
