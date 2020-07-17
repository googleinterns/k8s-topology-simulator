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

func TestSimulate(t *testing.T) {
	// Happy path: 3 zones with normal values
	// Invalid inputs: zero length zoneInfos/endpointslices
	testCases := []struct {
		name          string
		testInputs    []Zone
		expectResults Stat
		expectErr     error
	}{
		{
			name: "Happy path",
			testInputs: []Zone{
				Zone{Nodes: 30, Endpoints: 60, Name: "a"},
				Zone{Nodes: 35, Endpoints: 70, Name: "b"},
				Zone{Nodes: 50, Endpoints: 80, Name: "c"},
			},
			expectResults: Stat{
				InZoneTraffic: 0.89,
				Workload: map[string]float64{
					"a": 0.98,
					"b": 1,
					"c": 1,
				},
				TrafficDetail: map[string]Traffic{
					"a": Traffic{
						IncomingTraffic: 0.28,
						OutgoingTraffic: map[string]float64{
							"a": 0.89,
							"b": 0.1,
						},
					},
					"b": Traffic{
						IncomingTraffic: 0.33,
						OutgoingTraffic: map[string]float64{
							"a": 0.07,
							"b": 0.92,
						},
					},
					"c": Traffic{
						IncomingTraffic: 0.38,
						OutgoingTraffic: map[string]float64{
							"a": 0.05,
							"b": 0.06,
							"c": 0.87,
						},
					},
				},
			},
			expectErr: nil,
		},
		{
			name:          "Invalid inputs",
			testInputs:    []Zone{},
			expectResults: Stat{},
			expectErr:     errors.New("Can't evaluate probability based on empty zones or endpointslices"),
		},
	}
	// omit the error, trust this method (tested other places)
	alg, _ := CreateAlg(0.4, 100)
	sim := TheoreticalSimulator{}
	for _, testcase := range testCases {
		t.Run(testcase.name, func(t *testing.T) {
			// omit the error, we assume the result is valid or we intentionally
			// make some mistakes here
			zoneinfo, _ := createZoneinfos(testcase.testInputs)
			slices, _ := alg.CreateSlices(zoneinfo)
			stat, err := sim.Simulate(zoneinfo, slices)
			if !reflect.DeepEqual(err, testcase.expectErr) {
				t.Errorf("[Theoretial Simulation] Got err: %v, expected err: %v", err, testcase.expectErr)
				return
			}
			if !compareTwoStat(stat, testcase.expectResults) {
				t.Errorf("[Theoretial Simulation] Got stat: %+v, expected stat: %+v", stat, testcase.expectResults)
			}
		})
	}
}

// Compare values of two stats that within some range, we assume they are equal
// for test use only
func compareTwoStat(s1, s2 Stat) bool {
	if len(s1.TrafficDetail) != len(s2.TrafficDetail) {
		return false
	}
	// Compare the percentage value rounded to int
	if int(s1.InZoneTraffic*100) != int(s2.InZoneTraffic*100) {
		return false
	}
	for index := range s1.Workload {
		// Compare the percentage value rounded to int
		if int(s1.Workload[index]*100) != int(s2.Workload[index]*100) {
			return false
		}
	}
	for zone := range s1.TrafficDetail {
		// Compare the percentage value rounded to int
		if int(s1.TrafficDetail[zone].IncomingTraffic*100) != int(s2.TrafficDetail[zone].IncomingTraffic*100) {
			return false
		}
		for index := range s1.TrafficDetail[zone].OutgoingTraffic {
			// Compare the percentage value rounded to int
			if int(s1.TrafficDetail[zone].OutgoingTraffic[index]*100) != int(s2.TrafficDetail[zone].OutgoingTraffic[index]*100) {
				return false
			}
		}
	}
	return true
}
