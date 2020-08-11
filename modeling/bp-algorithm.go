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
	"fmt"
	"math"
)

type BackPropagationAlgorithm struct {
	// inZoneCoeff is inZoneTrafficScoreWeight
	inZoneCoeff float64
	// devCoeff is deviationScoreWeight
	devCoeff    float64
	// maxRound is the total rounds of gradient ascent
	maxRound    int
	// useL2Norm indicates whether to use L2-norm (square sum) for deviation score
	// otherwise, L1-norm (abs sum) will be used
	useL2Norm   bool
}

type bpArgs struct {
	// n is the number of zones
	n int
	// r is the ratio of egress traffic for every zone
	// subject to: sum_{i=0}^{n-1} r[i] = 1.0
	r []float64
	// e is the ratio of endpoints for every zone
	// subject to: sum_{i=0}^{n-1} e[i] = 1.0
	e []float64
	// names are the names of zones
	names []string
}

// TODO:
// 1. Verify if my construction of slice groups result in the same zone-to-zone traffic as a[i][i] indicates.
// 2. Figure out why sometimes there occurs minus scores
// 3. Figure out if the score is different from the formula used by the simulator
// 4. Sometimes a[i][j] goes below 0

const (
	// alpha is the learning rate of gradient ascent
	alpha = 0.05
	// eps is epsilon, the numeric precision const
	eps = 1e-10
)

func (alg BackPropagationAlgorithm) CreateSliceGroups(region regionInfo) (ret map[string]EndpointSliceGroup, err error) {
	arg, a := alg.initArgs(region)
	bestA := a
	bestScore := alg.calcScore(arg, a)

	// Back propagation / gradient ascent
	beta := alpha
	for m := 0; m < alg.maxRound; m++ {
		d := alg.calcDerivation(arg, a)
		for i := 0; i < arg.n; i++ {
			// a[i][arg.n-1] is hard constrained: a[i][n-1] = (1 - a[i][0] - ... - a[i][n-2])
			// I think in this simple case, a hard constraint is better than soft constraint like Lagrange condition
			a[i][arg.n-1] = 1.0
			for j := 0; j < arg.n-1; j++ {
				a[i][j] += beta * d[i][j]
				a[i][arg.n-1] -= a[i][j]
			}

			// If some value <0, take the projection
			for j := 0; j < arg.n-1; j++ {
				if a[i][j] < 0 {
					a[i][arg.n-1] += a[i][j]
					a[i][j] = 0
				}
			}
			if a[i][arg.n-1] < 0 {
				for {
					nonZero := 0
					min := math.MaxFloat64
					for j := 0; j < arg.n-1; j++ {
						if a[i][j] > eps {
							min = math.Min(a[i][j], min)
							nonZero ++
						}
					}
					val := - a[i][arg.n-1] / float64(nonZero)
					flag := false
					if min >= val {
						flag = true
					} else {
						val = min
					}
					for j := 0; j < arg.n-1; j++ {
						if a[i][j] > eps {
							a[i][j] -= val
							a[i][arg.n-1] += val
						}
					}
					if flag {
						break
					}
				}
				a[i][arg.n-1] = 0
			}
		}
		score := alg.calcScore(arg, a)
		if score > bestScore {
			bestA = a
			bestScore = score
		}
		// Let the real learning rate be decreasing to make it converge
		// Seems not to work well, but no damage
		beta = beta * 0.99
	}

	// Create slices
	// This works as long as every zone has >1 endpoints
	ret = make(map[string]EndpointSliceGroup)
	for i := 0; i < arg.n; i++ {
		name := arg.names[i]
		zone := region.zoneDetails[name]
		tot := zone.Endpoints
		m := 0
		// Each group (i-th) only contains endpoints within a zone
		// With the ingress ZoneTrafficWeights (j-th) being a[j][i]
		for tot > 0 {
			cur := tot
			if cur > 100 {
				cur = 100
			}
			tot -= cur
			m += 1

			sgName := fmt.Sprintf("%s-%d", name, m)
			sg := EndpointSliceGroup{
				Label: sgName,
				Composition: map[string]weightedEndpoints{
					name: {
						weight: 1.0,
						number: cur,
					},
				},
				ZoneTrafficWeights: map[string]float64{},
			}
			sum := 0.0
			for j := 0; j < arg.n; j++ {
				sg.ZoneTrafficWeights[arg.names[j]] = bestA[j][i]
				sum += bestA[j][i]
			}
			if math.Abs(sum) > eps {
				for j := 0; j < arg.n; j++ {
					sg.ZoneTrafficWeights[arg.names[j]] /= sum
				}
			}
			ret[sgName] = sg
		}
	}
	return
}

func (alg BackPropagationAlgorithm) initArgs(region regionInfo) (arg bpArgs, a [][]float64) {
	arg.n = len(region.zoneDetails)
	arg.r = make([]float64, arg.n)
	arg.e = make([]float64, arg.n)
	arg.names = make([]string, arg.n)
	i := 0
	for name, zone := range region.zoneDetails {
		arg.r[i] = zone.nodesRatio
		arg.e[i] = zone.endpointsRatio
		arg.names[i] = name
		i++
	}

	// Init zone-to-zone traffic matrix
	// a[i][j] = how many traffic from zone-i is forwarded to zone-j (percentage over zone-i)
	// subject to: sum_{j=0}^{n-1} a[i][j] = 1.0 for all 0<=i<=n-1
	a = make([][]float64, arg.n)
	for i := 0; i < arg.n; i++ {
		a[i] = make([]float64, arg.n)
		a[i][i] = 1.0
	}
	return
}

func (alg BackPropagationAlgorithm) calcScore(arg bpArgs, a [][]float64) float64 {
	// I'm not sure if this yields the same score as the simulator
	inZoneScore := 0.0
	for i := 0; i < arg.n; i++ {
		inZoneScore += arg.r[i] * a[i][i]
	}
	devScore := 0.0
	for i := 0; i < arg.n; i++ {
		for j := 0; j < arg.n; j++ {
			if alg.useL2Norm {
				devScore += math.Pow(arg.r[i]/(arg.e[j]+eps)*a[i][j]-1.0, 2)
			} else {
				devScore += math.Abs(arg.r[i]/(arg.e[j]+eps)*a[i][j] - 1.0)
			}
		}
	}
	return alg.inZoneCoeff*inZoneScore - alg.devCoeff*devScore
}

func (alg BackPropagationAlgorithm) calcDerivation(arg bpArgs, a [][]float64) (d [][]float64) {
	d = make([][]float64, arg.n)
	for i := 0; i < arg.n; i++ {
		d[i] = make([]float64, arg.n)
		// Deviation score
		for j := 0; j < arg.n-1; j++ {
			c := arg.r[i] / (arg.e[j] + eps)
			if alg.useL2Norm {
				d[i][j] = - 2 * alg.devCoeff * c * (c * a[i][j] - 1)
			} else {
				if c*(a[i][j]+eps) > 1.0 + eps {
					d[i][j] = - alg.devCoeff * c
				} else if c*(a[i][j]+eps) < 1.0 - eps {
					d[i][j] = alg.devCoeff * c
				}
			}
		}

		// The last one is constrained: a[i][n-1] = (1 - a[i][0] - ... - a[i][n-2])
		for j := arg.n - 1; j < arg.n; j++ {
			c := arg.r[i] / (arg.e[j] + eps)
			if alg.useL2Norm {
				for k := 0; k < j; k++ {
					d[i][k] += 2 * alg.devCoeff * c * (c * a[i][j] - 1)
				}
			}else{
				if c*(a[i][j]+eps) > 1.0 + eps {
					for k := 0; k < j; k++ {
						d[i][k] += alg.devCoeff * c
					}
				} else if c*(a[i][j]+eps) < 1.0 - eps {
					for k := 0; k < j; k++ {
						d[i][k] -= alg.devCoeff * c
					}
				}
			}
		}

		// In-zone score
		if i < arg.n-1 {
			d[i][i] += alg.inZoneCoeff * arg.r[i]
		} else {
			for k := 0; k < i; k++ {
				d[i][k] -= alg.inZoneCoeff * arg.r[i]
			}
		}
	}
	return
}
