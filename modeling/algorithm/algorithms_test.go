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
	"math"
	"reflect"
	"testing"

	"github.com/googleinterns/k8s-topology-simulator/modeling/types"
)

type algTestCase struct {
	name           string
	input          []types.Zone
	expectedOutput map[string]types.EndpointSliceGroup
	expectedErr    error
}
type routingAlgorithmTest struct {
	algName   string
	alg       RoutingAlgorithm
	testCases []algTestCase
}

func (algTest routingAlgorithmTest) doTest(t *testing.T) {
	for _, testcase := range algTest.testCases {
		t.Run(testcase.name, func(t *testing.T) {
			region, err := types.CreateRegionInfo(testcase.input)
			if err != nil {
				t.Errorf("[test %s] encountered unexpected error while creating RegionInfo with %+v", algTest.algName, testcase.input)
				return
			}
			sliceGroups, err := algTest.alg.CreateSliceGroups(region)
			if !reflect.DeepEqual(err, testcase.expectedErr) {
				t.Errorf("[test %s] got error: %v, expected err: %v", algTest.algName, err, testcase.expectedErr)
				return
			}
			if !deepCompareSliceGroups(t, sliceGroups, testcase.expectedOutput) {
				t.Errorf("[test %s] got slices: %+v, expected slices: %+v", algTest.algName, sliceGroups, testcase.expectedOutput)
				return
			}
		})
	}
}

// helper function to compare float64 that if two floats are within epsilon
// delta, we deem them as equal
func compareFloat(a float64, b float64, epsilon float64) bool {
	return math.Abs(a-b) <= epsilon
}

// deep compare two sliceGroups in two directions.
func deepCompareSliceGroups(t *testing.T, sliceGroupsA map[string]types.EndpointSliceGroup, sliceGroupsB map[string]types.EndpointSliceGroup) bool {
	// do a two-direction comparison to make sure the keys are equivalent. i.e.
	// mapA: key1: v1, key2: v2. mapB: key1: v1 will return equal under one
	// direction comparasion.
	t.Helper()
	return compareSliceGroups(t, sliceGroupsA, sliceGroupsB) && compareSliceGroups(t, sliceGroupsB, sliceGroupsA)
}

// helper function to compare EndpointSliceGroups. The zero value will be
// considered equivalent to the key not being set.
func compareSliceGroups(t *testing.T, sliceGroupsA map[string]types.EndpointSliceGroup, sliceGroupsB map[string]types.EndpointSliceGroup) bool {
	t.Helper()
	if (sliceGroupsA == nil) != (sliceGroupsB == nil) {
		t.Logf("expected two sliceGroups to be both nil or both non-nil, got %+v, %+v", sliceGroupsA, sliceGroupsB)
		return false
	}
	// deep comparison on every key/value pair
	for key, sliceGroupB := range sliceGroupsB {
		sliceGroupA := sliceGroupsA[key]
		if sliceGroupA.Label != sliceGroupB.Label {
			t.Logf("expected two sliceGroup with same label under key %s, got %s, %s",
				key, sliceGroupA.Label, sliceGroupB.Label)
			return false
		}
		if (sliceGroupA.Composition == nil) != (sliceGroupB.Composition == nil) {
			t.Logf("expected two sliceGroup.Composition to be both nil or both non-nil, got %s: %+v, %s: %+v",
				sliceGroupA.Label, sliceGroupA.Composition, sliceGroupB.Label, sliceGroupB.Composition)
			return false
		}
		for zone, compB := range sliceGroupB.Composition {
			compA := sliceGroupA.Composition[zone]
			if compA.Number != compB.Number {
				t.Logf("expected two compositions with same contribution from %s, got %+v for %s, %+v for %s",
					zone, compA, sliceGroupA.Label, compB, sliceGroupB.Label)
				return false
			}
			// if number = 0, weight doesn't matter
			if !compareFloat(compA.Weight, compB.Weight, 0.00001) {
				if compA.Number == 0 && compB.Number == 0 {
					return true
				}
				t.Logf("expected two compositions with same contribution from %s, got %+v for %s, %+v for %s",
					zone, compA, sliceGroupA.Label, compB, sliceGroupB.Label)
				return false
			}
		}
		if (sliceGroupA.ZoneTrafficWeights == nil) != (sliceGroupB.ZoneTrafficWeights == nil) {
			t.Logf("expected two sliceGroup.ZoneTrafficWeights to be both nil or both non-nil, got %s: %+v, %s: %+v",
				sliceGroupA.Label, sliceGroupA.ZoneTrafficWeights, sliceGroupB.Label, sliceGroupB.ZoneTrafficWeights)
			return false
		}
		for zone, weightB := range sliceGroupB.ZoneTrafficWeights {
			weightA := sliceGroupA.ZoneTrafficWeights[zone]
			if !compareFloat(weightA, weightB, 0.00001) {
				t.Logf("expected two sliceGroups with same routing weights towards %s, got %v from %s, %v from %s",
					zone, weightA, sliceGroupA.Label, weightB, sliceGroupB.Label)
				return false
			}
		}
	}
	return true
}
