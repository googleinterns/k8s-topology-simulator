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
	"reflect"
	"testing"
)

// Use one mock simulator to create dummy stat
// Use two mock algorithms to create slices:
//  mock1: return single slicegroup size > 100
//  mock2: return single slicegroup size <= 100

type testParam struct {
	zones []Zone
	alg   RoutingAlgorithm
	sim   TrafficSimulator
}

func TestNewModel(t *testing.T) {
	// Happy path: mock alg, mock simulator, normal zones -- verify slices
	// Invalid zones: nil zones -- verify error
	// Invalid alg: nil poitner -- verify error
	// Invalid simulator: nil pointer -- verify error
	testCases := []struct {
		name          string
		testInputs    testParam
		expectResults map[string]EndpointSliceGroup
		expectErr     error
	}{
		{
			name: "Happy path",
			testInputs: testParam{
				zones: []Zone{
					Zone{Nodes: 10, Endpoints: 10},
					Zone{Nodes: 10, Endpoints: 10},
				},
				alg: &MockAlg1{},
				sim: &MockSimulator{},
			},
			expectResults: map[string]EndpointSliceGroup{
				"a": EndpointSliceGroup{
					Composition: map[string]int{"a": 200},
				},
			},
			expectErr: nil,
		},
		{
			name: "Invalid zones",
			testInputs: testParam{
				zones: []Zone{},
				alg:   &MockAlg1{},
				sim:   &MockSimulator{},
			},
			expectResults: nil,
			expectErr:     errors.New("Creating zoneinfos with zero length []Zone"),
		},
		{
			name: "Invalid alg",
			testInputs: testParam{
				zones: []Zone{
					Zone{Nodes: 10, Endpoints: 10},
					Zone{Nodes: 10, Endpoints: 10},
				},
				alg: nil,
				sim: &MockSimulator{},
			},
			expectResults: nil,
			expectErr:     errors.New("Can't create model with nil algorithm or simulator"),
		},
		{
			name: "Invalid simulator",
			testInputs: testParam{
				zones: []Zone{
					Zone{Nodes: 10, Endpoints: 10},
					Zone{Nodes: 10, Endpoints: 10},
				},
				alg: &MockAlg1{},
				sim: nil,
			},
			expectResults: nil,
			expectErr:     errors.New("Can't create model with nil algorithm or simulator"),
		},
	}
	for _, testcase := range testCases {
		t.Run(testcase.name, func(t *testing.T) {
			model, err := NewModel(testcase.testInputs.zones, testcase.testInputs.alg, testcase.testInputs.sim)
			if !reflect.DeepEqual(err, testcase.expectErr) {
				t.Errorf("[Create Model] Got error: %v, expected err: %v", err, testcase.expectErr)
				return
			}
			if err != nil {
				// avoid access nil model pointer
				return
			}
			if !reflect.DeepEqual(model.slices, testcase.expectResults) {
				t.Errorf("[Create Model] Got slices: %+v, expected slices: %+v", model, testcase.expectResults)
				return
			}
		})
	}
}

func TestStartSimulation(t *testing.T) {
	// Happy path: mock alg, mocl simulator, normal zones -> normal model --
	// verify lastresult
	testCases := []struct {
		name          string
		testInputs    testParam
		expectResults Stat
		expectErr     error
	}{
		{
			name: "Happy path",
			testInputs: testParam{
				zones: []Zone{
					Zone{Nodes: 10, Endpoints: 10},
					Zone{Nodes: 10, Endpoints: 10},
				},
				alg: &MockAlg1{},
				sim: &MockSimulator{},
			},
			expectResults: Stat{InZoneTraffic: 1.0},
			expectErr:     nil,
		},
	}
	for _, testcase := range testCases {
		t.Run(testcase.name, func(t *testing.T) {
			model, err := NewModel(testcase.testInputs.zones, testcase.testInputs.alg, testcase.testInputs.sim)
			if err != nil {
				t.Errorf("[Model StartSimulation] Unexpected error encounted while creating model: %+v", err)
				return
			}
			err = model.StartSimulation()
			if !reflect.DeepEqual(err, testcase.expectErr) {
				t.Errorf("[Model StartSimulation] Got error: %v, expected err: %v", err, testcase.expectErr)
				return
			}
			if !reflect.DeepEqual(model.Results[0], testcase.expectResults) {
				t.Errorf("[Model StartSimulation] Got results: %+v, expected results: %+v", model, testcase.expectResults)
				return
			}
		})
	}
}

func TestGetEndpointslices(t *testing.T) {
	// Test slicegroup value, test number of slices
	testCases := []struct {
		name          string
		testInputs    testParam
		expectResults uint
		expectErr     error
	}{
		{
			name: "Slicegroup > 100",
			testInputs: testParam{
				zones: []Zone{
					Zone{Nodes: 10, Endpoints: 10},
					Zone{Nodes: 10, Endpoints: 10},
				},
				alg: &MockAlg1{},
				sim: &MockSimulator{},
			},
			expectResults: 2,
			expectErr:     nil,
		},
		{
			name: "Slicegroup < 100",
			testInputs: testParam{
				zones: []Zone{
					Zone{Nodes: 10, Endpoints: 10},
					Zone{Nodes: 10, Endpoints: 10},
				},
				alg: &MockAlg2{},
				sim: &MockSimulator{},
			},
			expectResults: 1,
			expectErr:     nil,
		},
	}
	for _, testcase := range testCases {
		t.Run(testcase.name, func(t *testing.T) {
			model, err := NewModel(testcase.testInputs.zones, testcase.testInputs.alg, testcase.testInputs.sim)
			if err != nil {
				t.Errorf("[Model GetEndpointSlices] Unexpected error encounted while creating model: %+v", err)
				return
			}
			// omit the first returned value which is trivial
			_, numberOfSlices, err := model.GetEndpointslices()
			if !reflect.DeepEqual(err, testcase.expectErr) {
				t.Errorf("[Model GetEndpointSlices] Got error: %v, expected err: %v", err, testcase.expectErr)
				return
			}
			if numberOfSlices != testcase.expectResults {
				t.Errorf("[Model GetEndpointSlices] Got number of slices: %v, expected number of slices: %v", numberOfSlices, testcase.expectResults)
				return
			}
		})
	}
}
