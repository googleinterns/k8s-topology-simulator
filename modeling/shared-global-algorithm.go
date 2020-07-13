package modeling

import (
	"errors"
	"math"
)

type SharedGlobalAlgorithm struct {
	globalWeight   float64
	globalTheshold int
}

func (alg SharedGlobalAlgorithm) CreatingSlices(zones []Zone) ([]Endpointslice, error) {
	var totalPods, totalNodes int
	tafficRatio := make([]float64, len(zones))
	capacityRatio := make([]float64, len(zones))
	deviation := make([]float64, len(zones))

	for _, zone := range zones {
		totalPods += zone.Endpoints
		totalNodes += zone.Nodes
	}

	for index, zone := range zones {
		tafficRatio[index] = float64(zone.Nodes) / float64(totalNodes)
		capacityRatio[index] = float64(zone.Endpoints) / float64(totalPods)
		deviation[index] = float64(zone.Endpoints) - float64(totalPods)*tafficRatio[index]
	}

	var endpointslices []Endpointslice
	var globalSlice Endpointslice
	globalSlice.Label = "global"
	globalSlice.Composition = make(map[string]int)
	globalSlice.Weights = make(map[string]float64)
	for index, zone := range zones {
		var globalPods int
		if totalPods <= alg.globalTheshold {
			globalPods = zone.Endpoints
		} else {
			globalPods = int(math.Min(math.Max(0.0, deviation[index])/alg.globalWeight, float64(zone.Endpoints)))
		}

		globalSlice.Composition[zone.Name] = globalPods
		globalSlice.Weights[zone.Name] = alg.globalWeight

		var slice Endpointslice
		slice.Label = zone.Name
		slice.Composition = make(map[string]int)
		slice.Weights = make(map[string]float64)
		slice.Composition[zone.Name] = zone.Endpoints - globalPods
		slice.Weights[zone.Name] = 1.0

		endpointslices = append(endpointslices, slice)
	}
	endpointslices = append(endpointslices, globalSlice)
	return endpointslices, nil
}

func CreateAlg(weight float64, threshold int) (SharedGlobalAlgorithm, error) {
	if weight < 0 || threshold < 0 {
		return SharedGlobalAlgorithm{}, errors.New("Invalid weight/threshold values to init algorihtm")
	}
	alg := SharedGlobalAlgorithm{globalWeight: weight, globalTheshold: threshold}
	return alg, nil
}
