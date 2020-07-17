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

func TestCreateZoneinfos(t *testing.T) {
	// happy path: zones 60-30, 70-35
	// invalid zones: zones 0-30, 10-20
	// invalid zones2: zones 30-0, 10-20
	// invalid zones3: zones -30-10, 10-20
	// invalid zones4: zones 30-(-10), 10-20
	// invalid []Zone: nil
	// invalid []Zone: empty slice
	testCases := []struct {
		name          string
		testInputs    []Zone
		expectResults zoneInfos
		expectErr     error
	}{
		{
			name: "Happy path",
			testInputs: []Zone{
				Zone{Nodes: 30, Endpoints: 60, Name: "a"},
				Zone{Nodes: 35, Endpoints: 70, Name: "b"},
			},
			expectResults: zoneInfos{
				totalNodes:     65,
				totalEndpoints: 130,
				zoneDetails: map[string]Zone{
					"a": Zone{Nodes: 30, Endpoints: 60, Name: "a", endpointsRatio: 60.0 / 130.0, nodesRatio: 30.0 / 65.0},
					"b": Zone{Nodes: 35, Endpoints: 70, Name: "b", endpointsRatio: 70.0 / 130.0, nodesRatio: 35.0 / 65.0},
				},
			},
			expectErr: nil,
		},
		{
			name: "Invalid zones",
			testInputs: []Zone{
				Zone{Nodes: 30, Endpoints: 0, Name: "a"},
				Zone{Nodes: 20, Endpoints: 10, Name: "b"},
			},
			expectResults: zoneInfos{},
			expectErr:     errors.New("Invalid zones with number of nodes or endpoints <= 0"),
		},
		{
			name: "Invalid zones2",
			testInputs: []Zone{
				Zone{Nodes: 0, Endpoints: 30, Name: "a"},
				Zone{Nodes: 20, Endpoints: 10, Name: "b"},
			},
			expectResults: zoneInfos{},
			expectErr:     errors.New("Invalid zones with number of nodes or endpoints <= 0"),
		},
		{
			name: "Invalid zones3",
			testInputs: []Zone{
				Zone{Nodes: 10, Endpoints: -30, Name: "a"},
				Zone{Nodes: 20, Endpoints: 10, Name: "b"},
			},
			expectResults: zoneInfos{},
			expectErr:     errors.New("Invalid zones with number of nodes or endpoints <= 0"),
		},
		{
			name: "Invalid zones4",
			testInputs: []Zone{
				Zone{Nodes: -30, Endpoints: 10, Name: "a"},
				Zone{Nodes: 20, Endpoints: 10, Name: "b"},
			},
			expectResults: zoneInfos{},
			expectErr:     errors.New("Invalid zones with number of nodes or endpoints <= 0"),
		},
		{
			name:          "Invalid []Zone",
			testInputs:    nil,
			expectResults: zoneInfos{},
			expectErr:     errors.New("Creating zoneinfos with zero length []Zone"),
		},
		{
			name:          "Invalid []Zone2",
			testInputs:    []Zone{},
			expectResults: zoneInfos{},
			expectErr:     errors.New("Creating zoneinfos with zero length []Zone"),
		},
	}
	for _, testcase := range testCases {
		t.Run(testcase.name, func(t *testing.T) {
			zoneInfo, err := createZoneinfos(testcase.testInputs)
			if !reflect.DeepEqual(err, testcase.expectErr) {
				t.Errorf("[Create zoneinfo] Got error: %v, expected err: %v", err, testcase.expectErr)
				return
			}
			if !reflect.DeepEqual(zoneInfo, testcase.expectResults) {
				t.Errorf("[Create zoneinfo] Got zoneinfo: %+v, expected zoneinfo: %+v", zoneInfo, testcase.expectResults)
				return
			}
		})
	}
}
