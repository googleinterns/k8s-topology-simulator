package modeling

import (
	"errors"
	"math"
)

type SharedGlobalAlgorithm struct {
	// Weight of global endpointslces
	globalWeight float64
	// Threshold of global endpointslces that if the total number of endpoints <= threshold, all endpoints go to global endpointslice
	//		Int should be enough
	globalTheshold int
}

func (alg SharedGlobalAlgorithm) CreatingSlices(zones []Zone) ([]Endpointslice, error) {
	// Total number of endpoints and nodes of all zones
	var totalPods, totalNodes int
	// This assumes the incoming traffic of a zone is proportional to the number of nodes of that zone
	tafficRatio := make([]float64, len(zones))
	// This assumes the capacity handling the requests of a zone is proportional to the number of endpoints in that zone
	capacityRatio := make([]float64, len(zones))
	// The deviation for the traffic and capacity above
	deviation := make([]float64, len(zones))

	for _, zone := range zones {
		if zone.Endpoints <= 0 || zone.Nodes <= 0 {
			return nil, errors.New("Can't create slices based on zone with either 0 endpoints or 0 nodes")
		}
		totalPods += zone.Endpoints
		totalNodes += zone.Nodes
	}

	for index, zone := range zones {
		// Calculating the metrics
		tafficRatio[index] = float64(zone.Nodes) / float64(totalNodes)
		capacityRatio[index] = float64(zone.Endpoints) / float64(totalPods)
		deviation[index] = float64(zone.Endpoints) - float64(totalPods)*tafficRatio[index]
	}

	// Output endpointslices
	var endpointslices []Endpointslice
	// The 'big' global slice -- might be split into many small global slices when the number of endpoints > required number of endpoints/endpoinslice, i.e. 100 for default
	//		not able to divide the big one into smaller ones that the contributions may vary and there is no need to do so either.
	var globalSlice Endpointslice
	globalSlice.Label = "global"
	globalSlice.Composition = make(map[string]int)
	globalSlice.Weights = make(map[string]float64)
	for index, zone := range zones {
		var globalPods int
		// If total pods <= threshold, all pods go to global slice
		if totalPods <= alg.globalTheshold {
			globalPods = zone.Endpoints
		} else {
			// Otherwise calculte the global contribution of current zone based on the global weight and the deviation of this zone
			globalPods = int(math.Min(math.Max(0.0, deviation[index])/alg.globalWeight, float64(zone.Endpoints)))
		}

		globalSlice.Composition[zone.Name] = globalPods
		globalSlice.Weights[zone.Name] = alg.globalWeight

		// Calculate how many endpoints remain in the local zone
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
