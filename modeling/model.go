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
)

// Model wrapper class for the simulation components
type Model struct {
	slices    map[string]EndpointSliceGroup
	alg       RoutingAlgorithm
	simulator TrafficSimulator
	region    regionInfo

	// SliceCapacity is the number of max endpoints per slice
	SliceCapacity int
}

// NewModel creates a model with routing algorithm and traffic simulator
func NewModel(alg RoutingAlgorithm, sim TrafficSimulator) (*Model, error) {
	if alg == nil || sim == nil {
		return nil, errors.New("can't create model with nil algorithm or simulator")
	}
	model := &Model{
		SliceCapacity: 100,
		alg:           alg,
		simulator:     sim,
	}
	return model, nil
}

// UpdateRegion updates the region of the model, this is used to run the
// algorithm on different zone inputs
func (m *Model) UpdateRegion(zones []Zone) error {
	region, err := createRegionInfo(zones)
	if err != nil {
		return err
	}
	slices, err := m.alg.CreateSliceGroups(region)
	if err != nil {
		return err
	}
	m.region = region
	m.slices = slices
	return nil
}

// StartSimulation based on the zones(Region) and EndpointSliceGroups
func (m *Model) StartSimulation() (SimulationResult, error) {
	return m.simulator.Simulate(m.region, m.slices)
}

// GetNumberOfEndpointSlices returns the number of EndpointSlices
func (m *Model) GetNumberOfEndpointSlices() int {
	totalSlices := 0
	for _, slice := range m.slices {
		endpoints := slice.numberOfEndpoints()
		totalSlices += endpoints / m.SliceCapacity
		if endpoints%m.SliceCapacity != 0 {
			totalSlices++
		}
	}
	return totalSlices
}

// GetNumberOfEndpoints returns the number of total endpoints
func (m *Model) GetNumberOfEndpoints() int {
	return m.region.totalEndpoints
}
