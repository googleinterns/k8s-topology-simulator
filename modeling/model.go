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

	Region  regionInfo
	Results []SimulationResult
	// SliceCapacity is the number of max endpoints per slice
	SliceCapacity uint
}

// NewModel creates a model with zones, routing algorithm and traffic simulator
// and uses the algorithm to create the EndpointSliceGroups
func NewModel(zones []Zone, alg RoutingAlgorithm, sim TrafficSimulator) (*Model, error) {
	region, err := createRegionInfo(zones)
	if err != nil {
		return nil, err
	}
	if alg == nil || sim == nil {
		return nil, errors.New("can't create model with nil algorithm or simulator")
	}
	model := &Model{
		Region:        region,
		SliceCapacity: 100,
		alg:           alg,
		simulator:     sim,
	}
	slices, err := model.alg.CreateSliceGroups(model.Region)
	if err != nil {
		return nil, err
	}
	model.slices = slices
	return model, nil
}

// StartSimulation based on the zones(Region) and EndpointSliceGroups
func (m *Model) StartSimulation() error {
	simResult, err := m.simulator.Simulate(m.Region, m.slices)
	if err != nil {
		return err
	}
	m.Results = append(m.Results, simResult)
	return nil
}

// PrintLastResult prints the summary of the last simulation result
func (m *Model) PrintLastResult() error {
	numberOfSlices := m.GetNumberOfEndpointSlices()
	lastResult := m.Results[len(m.Results)-1]
	fmt.Printf("%% in-zone traffic \t %.2f%%\n", lastResult.InZoneTraffic*100)
	fmt.Printf("# of endpoint slices\t %v\n", numberOfSlices)
	fmt.Printf("# of endpoints\t %d\n", m.Region.totalEndpoints)
	fmt.Println("----------------------------------------------")

	for zone, traffic := range lastResult.TrafficDetail {
		fmt.Printf("Total to %s \t %.f%% \n", zone, traffic.Incoming*100)
	}
	fmt.Println("----------------------------------------------")

	for zone, traffic := range lastResult.TrafficDetail {
		fmt.Printf("From %s : \n", zone)
		for z, t := range traffic.Outgoing {
			fmt.Printf("\t to %s : %.2f%% \n", z, t*100)
		}
	}
	fmt.Println("----------------------------------------------")

	for zone, workload := range lastResult.Workload {
		fmt.Printf("Workload for %s : \t %.2f%% \n", zone, workload*100)
	}
	return nil
}

// GetEndpointSliceGroups returns the EndpointSliceGroups, don't allow users to
// manually change this field -- use this getter to get the value
func (m *Model) GetEndpointSliceGroups() map[string]EndpointSliceGroup {
	return m.slices
}

// GetNumberOfEndpointSlices returns the number of EndpointSlices
func (m *Model) GetNumberOfEndpointSlices() uint {
	totalSlices := uint(0)
	for _, slice := range m.slices {
		pods := slice.numberOfPods()
		totalSlices += uint(pods) / m.SliceCapacity
		if uint(pods)%m.SliceCapacity != 0 {
			totalSlices++
		}
	}
	return totalSlices
}
