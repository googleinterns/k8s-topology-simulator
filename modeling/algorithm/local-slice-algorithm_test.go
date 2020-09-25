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
	"testing"

	"github.com/googleinterns/k8s-topology-simulator/modeling/types"
)

func TestLocalAlgorithm(t *testing.T) {
	testCases := []algTestCase{
		{
			name: "unbalanced nodes distribution",
			input: []types.Zone{
				types.Zone{
					Nodes:     1,
					Endpoints: 5,
					Name:      "ZoneA",
				},
				types.Zone{
					Nodes:     2,
					Endpoints: 20,
					Name:      "ZoneB",
				},
				types.Zone{
					Nodes:     7,
					Endpoints: 20,
					Name:      "ZoneC",
				},
			},
			expectedOutput: map[string]types.EndpointSliceGroup{
				"ZoneA": types.EndpointSliceGroup{
					Label: "ZoneA",
					Composition: map[string]types.WeightedEndpoints{
						"ZoneA": types.WeightedEndpoints{Number: 5, Weight: 1},
					},
					ZoneTrafficWeights: map[string]float64{
						"ZoneA": 1,
					},
				},
				"ZoneB": types.EndpointSliceGroup{
					Label: "ZoneB",
					Composition: map[string]types.WeightedEndpoints{
						"ZoneB": types.WeightedEndpoints{Number: 9, Weight: 1},
					},
					ZoneTrafficWeights: map[string]float64{
						"ZoneB": 1,
					},
				},
				"ZoneC": types.EndpointSliceGroup{
					Label: "ZoneC",
					Composition: map[string]types.WeightedEndpoints{
						"ZoneB": types.WeightedEndpoints{Number: 11, Weight: 1},
						"ZoneC": types.WeightedEndpoints{Number: 20, Weight: 1},
					},
					ZoneTrafficWeights: map[string]float64{
						"ZoneC": 1,
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "corner case : zero endpoints",
			input: []types.Zone{
				types.Zone{
					Nodes:     1,
					Endpoints: 0,
					Name:      "ZoneA",
				},
				types.Zone{
					Nodes:     1,
					Endpoints: 6,
					Name:      "ZoneB",
				},
				types.Zone{
					Nodes:     1,
					Endpoints: 7,
					Name:      "ZoneC",
				},
			},
			expectedOutput: map[string]types.EndpointSliceGroup{
				"ZoneA": types.EndpointSliceGroup{
					Label: "ZoneA",
					Composition: map[string]types.WeightedEndpoints{
						"ZoneB": types.WeightedEndpoints{Number: 1, Weight: 1},
						"ZoneC": types.WeightedEndpoints{Number: 2, Weight: 1},
					},
					ZoneTrafficWeights: map[string]float64{
						"ZoneA": 1,
					},
				},
				"ZoneB": types.EndpointSliceGroup{
					Label: "ZoneB",
					Composition: map[string]types.WeightedEndpoints{
						"ZoneB": types.WeightedEndpoints{Number: 5, Weight: 1},
					},
					ZoneTrafficWeights: map[string]float64{
						"ZoneB": 1,
					},
				},
				"ZoneC": types.EndpointSliceGroup{
					Label: "ZoneC",
					Composition: map[string]types.WeightedEndpoints{
						"ZoneC": types.WeightedEndpoints{Number: 5, Weight: 1},
					},
					ZoneTrafficWeights: map[string]float64{
						"ZoneC": 1,
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "corner case : give out more endpoints",
			input: []types.Zone{
				types.Zone{
					Nodes:     16,
					Endpoints: 5,
					Name:      "ZoneA",
				},
				types.Zone{
					Nodes:     8,
					Endpoints: 1,
					Name:      "ZoneB",
				},
				types.Zone{
					Nodes:     1,
					Endpoints: 0,
					Name:      "ZoneC",
				},
			},
			expectedOutput: map[string]types.EndpointSliceGroup{
				"ZoneA": types.EndpointSliceGroup{
					Label: "ZoneA",
					Composition: map[string]types.WeightedEndpoints{
						"ZoneA": types.WeightedEndpoints{Number: 3, Weight: 1},
					},
					ZoneTrafficWeights: map[string]float64{
						"ZoneA": 1,
					},
				},
				"ZoneB": types.EndpointSliceGroup{
					Label: "ZoneB",
					Composition: map[string]types.WeightedEndpoints{
						"ZoneA": types.WeightedEndpoints{Number: 1, Weight: 1},
						"ZoneB": types.WeightedEndpoints{Number: 1, Weight: 1},
					},
					ZoneTrafficWeights: map[string]float64{
						"ZoneB": 1,
					},
				},
				"ZoneC": types.EndpointSliceGroup{
					Label: "ZoneC",
					Composition: map[string]types.WeightedEndpoints{
						"ZoneA": types.WeightedEndpoints{Number: 1, Weight: 1},
					},
					ZoneTrafficWeights: map[string]float64{
						"ZoneC": 1,
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "2 zones with no endpoints",
			input: []types.Zone{
				types.Zone{
					Nodes:     30,
					Endpoints: 100,
					Name:      "ZoneA",
				},
				types.Zone{
					Nodes:     30,
					Endpoints: 0,
					Name:      "ZoneB",
				},
				types.Zone{
					Nodes:     30,
					Endpoints: 0,
					Name:      "ZoneC",
				},
			},
			expectedOutput: map[string]types.EndpointSliceGroup{
				"ZoneA": types.EndpointSliceGroup{
					Label: "ZoneA",
					Composition: map[string]types.WeightedEndpoints{
						"ZoneA": types.WeightedEndpoints{Number: 34, Weight: 1},
					},
					ZoneTrafficWeights: map[string]float64{
						"ZoneA": 1,
					},
				},
				"ZoneB": types.EndpointSliceGroup{
					Label: "ZoneB",
					Composition: map[string]types.WeightedEndpoints{
						"ZoneA": types.WeightedEndpoints{Number: 33, Weight: 1},
					},
					ZoneTrafficWeights: map[string]float64{
						"ZoneB": 1,
					},
				},
				"ZoneC": types.EndpointSliceGroup{
					Label: "ZoneC",
					Composition: map[string]types.WeightedEndpoints{
						"ZoneA": types.WeightedEndpoints{Number: 33, Weight: 1},
					},
					ZoneTrafficWeights: map[string]float64{
						"ZoneC": 1,
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "only 1 endpoint",
			input: []types.Zone{
				types.Zone{
					Nodes:     30,
					Endpoints: 1,
					Name:      "ZoneA",
				},
				types.Zone{
					Nodes:     30,
					Endpoints: 0,
					Name:      "ZoneB",
				},
				types.Zone{
					Nodes:     30,
					Endpoints: 0,
					Name:      "ZoneC",
				},
			},
			expectedOutput: map[string]types.EndpointSliceGroup{
				"global": types.EndpointSliceGroup{
					Label: "global",
					Composition: map[string]types.WeightedEndpoints{
						"ZoneA": types.WeightedEndpoints{Number: 1, Weight: 1},
					},
					ZoneTrafficWeights: map[string]float64{
						"ZoneA": 1,
						"ZoneB": 1,
						"ZoneC": 1,
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "mostly balanced small",
			input: []types.Zone{
				types.Zone{
					Nodes:     1,
					Endpoints: 3,
					Name:      "ZoneA",
				},
				types.Zone{
					Nodes:     2,
					Endpoints: 2,
					Name:      "ZoneB",
				},
				types.Zone{
					Nodes:     2,
					Endpoints: 2,
					Name:      "ZoneC",
				},
			},
			expectedOutput: map[string]types.EndpointSliceGroup{
				"ZoneA": types.EndpointSliceGroup{
					Label: "ZoneA",
					Composition: map[string]types.WeightedEndpoints{
						"ZoneA": types.WeightedEndpoints{Number: 3, Weight: 1},
					},
					ZoneTrafficWeights: map[string]float64{
						"ZoneA": 1,
					},
				},
				"ZoneB": types.EndpointSliceGroup{
					Label: "ZoneB",
					Composition: map[string]types.WeightedEndpoints{
						"ZoneB": types.WeightedEndpoints{Number: 2, Weight: 1},
					},
					ZoneTrafficWeights: map[string]float64{
						"ZoneB": 1,
					},
				},
				"ZoneC": types.EndpointSliceGroup{
					Label: "ZoneC",
					Composition: map[string]types.WeightedEndpoints{
						"ZoneC": types.WeightedEndpoints{Number: 2, Weight: 1},
					},
					ZoneTrafficWeights: map[string]float64{
						"ZoneC": 1,
					},
				},
			},
			expectedErr: nil,
		},
	}
	localTest := routingAlgorithmTest{
		algName:   "LocalSlice",
		alg:       LocalSliceAlgorithm{threshold: 0.5},
		testCases: testCases,
	}
	localTest.doTest(t)
}
