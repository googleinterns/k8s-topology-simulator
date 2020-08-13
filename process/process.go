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

package process

import (
	"fmt"

	"github.com/googleinterns/k8s-topology-simulator/modeling"
	"github.com/googleinterns/k8s-topology-simulator/modeling/algorithm"
	"github.com/googleinterns/k8s-topology-simulator/modeling/simulator"
	"github.com/googleinterns/k8s-topology-simulator/modeling/types"
)

const endpointsPerSlice = 100
const inZoneTrafficScoreWeight, deviationScoreWeight, sliceScoreWeight = 0.4, 0.4, 0.2

// StartProcessing starts parsing input file, running simulation and
// generating output file
func StartProcessing(inputFile string, outputFile string, alg string) error {
	inputArray, err := parseInput(inputFile)
	if err != nil {
		return err
	}

	outputArray, err := startSimulation(alg, inputArray)
	if err != nil {
		return err
	}

	err = parseResult(outputFile, outputArray)
	return err
}

// every row of the input file will be parsed to one instance of inputData
type inputData struct {
	// input id of the row
	name string
	// parse zone info of the input file into zone data structure
	zones []types.Zone
}

// every instance of inputData will be mapped to one instance of outputData
type outputData struct {
	// same id as input id
	name string
	// number of endpoints associated with the input data
	endpoints int
	// number of EndpointSlices associated with the input data
	endpointSlices int
	// simulation result of that piece of input data
	result types.SimulationResult
}

// startSimulation processes simulation on input data, produces instances of
// outputData structure and returns a slice of them
func startSimulation(algName string, inputArray []inputData) ([]outputData, error) {
	// create algrithm based on the algorithm name delivered by the flag
	alg := algorithm.NewAlgorithm(algName)
	// create simulation model, currently do calculation based on probability
	// rather than real simulation.
	model, err := modeling.NewModel(alg, simulator.TheoreticalSimulator{})
	if err != nil {
		return nil, err
	}
	model.SliceCapacity = endpointsPerSlice
	var outputArray []outputData
	for _, rowData := range inputArray {
		err := model.UpdateRegion(rowData.zones)
		if err != nil {
			return outputArray, fmt.Errorf("error updating region for input : %s, %v", rowData.name, err)
		}
		simRes, err := model.StartSimulation()
		if err != nil {
			return outputArray, fmt.Errorf("error starting simulation for input : %s, %v", rowData.name, err)
		}
		outputArray = append(outputArray, outputData{name: rowData.name,
			endpoints:      model.GetNumberOfEndpoints(),
			endpointSlices: model.GetNumberOfEndpointSlices(),
			result:         simRes,
		})
	}

	return outputArray, nil
}
