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
	"math"
	"testing"
)

func TestDerivation(t *testing.T) {
	const (
		diff = 1e-6
		eps  = 1e-4
	)

	testCases := []struct {
		name      string
		useL2Norm bool
	}{
		{
			name:      "Test L2 Norm",
			useL2Norm: true,
		},
		{
			name:      "Test L1 Norm",
			useL2Norm: true,
		},
	}

	for _, testcase := range testCases {
		t.Run(testcase.name, func(t *testing.T) {
			alg := BackPropagationAlgorithm{
				inZoneCoeff: 0.5,
				devCoeff:    0.3,
				useL2Norm:   testcase.useL2Norm,
			}

			arg := bpArgs{
				n: 3,
				r: []float64{0.5, 0.3, 0.2},
				e: []float64{0.25, 0.6, 0.15},
			}

			a := [][]float64{
				{0.2, 0.5, 0.3},
				{0.1, 0.0, 0.9},
				{0.4, 0.2, 0.4},
			}
			baseScore := alg.calcScore(arg, a)
			d := alg.calcDerivation(arg, a)

			for i := 0; i < arg.n; i++ {
				for j := 0; j < arg.n-1; j++ {
					a[i][j] += diff
					a[i][arg.n-1] -= diff
					newScore := alg.calcScore(arg, a)
					a[i][j] -= diff
					a[i][arg.n-1] += diff

					deri := (newScore - baseScore) / diff
					if math.Abs(deri-d[i][j]) > eps {
						t.Errorf("Derivation at a[%d][%d] is wrong: expected %f, got %f", i, j, deri, d[i][j])
					}
				}
			}
		})
	}
}
