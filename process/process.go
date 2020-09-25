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
	"github.com/googleinterns/k8s-topology-simulator/modeling"
	"github.com/googleinterns/k8s-topology-simulator/modeling/algorithm"
	"github.com/googleinterns/k8s-topology-simulator/modeling/simulator"
	"github.com/googleinterns/k8s-topology-simulator/modeling/types"
	"k8s.io/klog/v2"
)

const endpointsPerSlice = 100
const inZoneTrafficScoreWeight, deviationScoreWeight, sliceScoreWeight = 0.45, 0.4, 0.15

// StartProcessing starts parsing input file, running simulation and
// generating output file
func StartProcessing(inputFile string, outputFile string, alg string) error {

	// initialize a goroutine to read row data from input file and put the
	// converted row data into a queue
	inputQueue, err := parseInput(inputFile)
	if err != nil {
		return err
	}

	// initialize a goroutine to process row data from inputQueue and put the
	// processed data into another queue to handle results
	outputQueue, err := startSimulation(alg, inputQueue)
	if err != nil {
		return err
	}

	// parse results from outputQueue and write to output file
	return parseResult(outputFile, outputQueue)
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
// outputData structure and puts them in a queue(channel)
func startSimulation(algName string, inputQueue <-chan inputData) (<-chan outputData, error) {
	// create algorithm based on the algorithm name
	alg := algorithm.NewAlgorithm(algName)
	// create simulation model, currently do calculation based on probability
	// rather than real simulation.
	model, err := modeling.NewModel(alg, simulator.TheoreticalSimulator{})
	if err != nil {
		return nil, err
	}
	outputQueue := make(chan outputData)
	// Some simplifications here result in this code not being threadsafe.
	// Do not use more than one goroutine to process this queue.
	go func() {
		defer close(outputQueue)

		for rowData, more := <-inputQueue; more; rowData, more = <-inputQueue {
			oData, rerr := runSimulation(model, rowData)
			if rerr == nil {
				outputQueue <- oData
			}
		}
	}()

	return outputQueue, err
}

// helper function helps to generate one piece of outputData from one piece of
// inputData
func runSimulation(model *modeling.Model, rowData inputData) (outputData, error) {
	err := model.UpdateRegion(rowData.zones)
	if err != nil {
		klog.Errorf("error updating region for input : %s, %v", rowData.name, err)
		return outputData{}, err
	}
	simRes, err := model.StartSimulation()
	if err != nil {
		klog.Errorf("error starting simulation for input : %s, %v", rowData.name, err)
		return outputData{}, err
	}
	return outputData{name: rowData.name,
		endpoints:      model.GetNumberOfEndpoints(),
		endpointSlices: model.GetNumberOfEndpointSlices(),
		result:         simRes}, nil
}
