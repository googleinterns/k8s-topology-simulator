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

type algorithmParam struct {
	weight    float64
	threshold int
}

func TestCreateAlg(t *testing.T) {
	// Happy path: normal cases
	// Happy path2: normal case weight = 0
	// Happy path3: normal case threshold = 0
	// Invalid weight: weight < 0
	// Invalid threshold : threshold < 0
	testCases := []struct {
		name          string
		testInputs    algorithmParam
		expectResults SharedGlobalAlgorithm
		expectErr     error
	}{
		{
			name: "Happy path",
			testInputs: algorithmParam{
				weight:    1.0,
				threshold: 100,
			},
			expectResults: SharedGlobalAlgorithm{globalWeight: 1.0, globalThreshold: 100},
			expectErr:     nil,
		},
		{
			name: "Happy path2",
			testInputs: algorithmParam{
				weight:    0.0,
				threshold: 100,
			},
			expectResults: SharedGlobalAlgorithm{globalWeight: 0.0, globalThreshold: 100},
			expectErr:     nil,
		},
		{
			name: "Happy path3",
			testInputs: algorithmParam{
				weight:    10.0,
				threshold: 0,
			},
			expectResults: SharedGlobalAlgorithm{globalWeight: 10.0, globalThreshold: 0},
			expectErr:     nil,
		},
		{
			name: "Invalid weight",
			testInputs: algorithmParam{
				weight:    -10.0,
				threshold: 10,
			},
			expectResults: SharedGlobalAlgorithm{},
			expectErr:     errors.New("Invalid weight/threshold values to init algorithm"),
		},
		{
			name: "Invalid threshold",
			testInputs: algorithmParam{
				weight:    10.0,
				threshold: -10,
			},
			expectResults: SharedGlobalAlgorithm{},
			expectErr:     errors.New("Invalid weight/threshold values to init algorithm"),
		},
	}
	for _, testcase := range testCases {
		t.Run(testcase.name, func(t *testing.T) {
			alg, err := CreateAlg(testcase.testInputs.weight, testcase.testInputs.threshold)
			if !reflect.DeepEqual(err, testcase.expectErr) {
				t.Errorf("[SharedGlobalAlgorithm CreateAlg] Got error: %v, expected err: %v", err, testcase.expectErr)
				return
			}
			if alg != testcase.expectResults {
				t.Errorf("[SharedGlobalAlgorithm CreateAlg] Got alg: %+v, expected alg: %+v", alg, testcase.expectResults)
				return
			}
		})
	}
}

func TestCreateSlices(t *testing.T) {
	// Happy path: 3 zones: 60-30, 70-35, 80-50
	// Happy path2: 3 zones: 80-10, 30-100, 90-15
	// Happy path3: 2 zones: 10-5, 10-5
	// Invalid zone: 1 zone: 0-10
	// Invalid zone2: 1 zone: 10-0
	testCases := []struct {
		name          string
		testInputs    []Zone
		expectResults map[string]EndpointSliceGroup
		expectErr     [2]error
	}{
		{
			name: "Happy path",
			testInputs: []Zone{
				Zone{Nodes: 30, Endpoints: 60, Name: "a"},
				Zone{Nodes: 35, Endpoints: 70, Name: "b"},
				Zone{Nodes: 50, Endpoints: 80, Name: "c"},
			},
			expectResults: map[string]EndpointSliceGroup{
				"global": EndpointSliceGroup{Label: "global", Composition: map[string]int{"a": 13, "b": 15, "c": 0}, Weights: map[string]float64{"a": 0.4, "b": 0.4, "c": 0.4}},
				"a":      EndpointSliceGroup{Label: "a", Composition: map[string]int{"a": 47}, Weights: map[string]float64{"a": 1}},
				"b":      EndpointSliceGroup{Label: "b", Composition: map[string]int{"b": 55}, Weights: map[string]float64{"b": 1}},
				"c":      EndpointSliceGroup{Label: "c", Composition: map[string]int{"c": 80}, Weights: map[string]float64{"c": 1}},
			},
			expectErr: [2]error{nil, nil},
		},
		{
			name: "Happy path2",
			testInputs: []Zone{
				Zone{Nodes: 10, Endpoints: 80, Name: "a"},
				Zone{Nodes: 100, Endpoints: 30, Name: "b"},
				Zone{Nodes: 15, Endpoints: 90, Name: "c"},
			},
			expectResults: map[string]EndpointSliceGroup{
				"global": EndpointSliceGroup{Label: "global", Composition: map[string]int{"a": 80, "b": 0, "c": 90}, Weights: map[string]float64{"a": 0.4, "b": 0.4, "c": 0.4}},
				"a":      EndpointSliceGroup{Label: "a", Composition: map[string]int{"a": 0}, Weights: map[string]float64{"a": 1}},
				"b":      EndpointSliceGroup{Label: "b", Composition: map[string]int{"b": 30}, Weights: map[string]float64{"b": 1}},
				"c":      EndpointSliceGroup{Label: "c", Composition: map[string]int{"c": 0}, Weights: map[string]float64{"c": 1}},
			},
			expectErr: [2]error{nil, nil},
		},
		{
			name: "Happy path3",
			testInputs: []Zone{
				Zone{Nodes: 5, Endpoints: 10, Name: "a"},
				Zone{Nodes: 5, Endpoints: 10, Name: "b"},
			},
			expectResults: map[string]EndpointSliceGroup{
				"global": EndpointSliceGroup{Label: "global", Composition: map[string]int{"a": 10, "b": 10}, Weights: map[string]float64{"a": 0.4, "b": 0.4}},
				"a":      EndpointSliceGroup{Label: "a", Composition: map[string]int{"a": 0}, Weights: map[string]float64{"a": 1}},
				"b":      EndpointSliceGroup{Label: "b", Composition: map[string]int{"b": 0}, Weights: map[string]float64{"b": 1}},
			},
			expectErr: [2]error{nil, nil},
		},
		{
			name: "Invalid zone",
			testInputs: []Zone{
				Zone{Nodes: 10, Endpoints: 0, Name: "a"},
			},
			expectResults: nil,
			expectErr:     [2]error{errors.New("Invalid zones with number of nodes or endpoints <= 0"), errors.New("Can't create endpointslices with 0 number of zone")},
		},
		{
			name: "Invalid zone2",
			testInputs: []Zone{
				Zone{Nodes: -10, Endpoints: 10, Name: "a"},
			},
			expectResults: nil,
			expectErr:     [2]error{errors.New("Invalid zones with number of nodes or endpoints <= 0"), errors.New("Can't create endpointslices with 0 number of zone")},
		},
		{
			name:          "Invalid zoneInfo",
			testInputs:    nil,
			expectResults: nil,
			expectErr:     [2]error{errors.New("Creating zoneinfos with zero length []Zone"), errors.New("Can't create endpointslices with 0 number of zone")},
		},
	}
	// omit the error, trust this method (tested above)
	alg, _ := CreateAlg(0.4, 100)
	for _, testcase := range testCases {
		t.Run(testcase.name, func(t *testing.T) {
			zones, err := createZoneinfos(testcase.testInputs)
			if !reflect.DeepEqual(err, testcase.expectErr[0]) {
				t.Errorf("[SharedGlobalAlgorithm CreateZoneinfos] Got error: %v, expected err: %v", err, testcase.expectErr[0])
				return
			}
			slices, err := alg.CreateSlices(zones)
			if !reflect.DeepEqual(err, testcase.expectErr[1]) {
				t.Errorf("[SharedGlobalAlgorithm CreateSlices] Got error: %v, expected err: %v", err, testcase.expectErr[1])
				return
			}
			if !reflect.DeepEqual(slices, testcase.expectResults) {
				t.Errorf("[SharedGlobalAlgorithm CreateSlices] Got slices: %+v, expected slices: %+v", slices, testcase.expectResults)
			}
		})
	}
}
