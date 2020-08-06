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

package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/googleinterns/k8s-topology-simulator/modeling"
	"k8s.io/klog/v2"
)

const endpointsPerSlice = 100
const inZoneTrafficScoreWeight, deviationScoreWeight, sliceScoreWeight = 0.5, 0.3, 0.2

func main() {
	// algorithm name, default shared global
	algPtr := flag.String("alg", "SharedGlobalAlgorithm", "routing algorithm")
	// input file
	inputPtr := flag.String("input", "example/input.csv", "inputs to use for this algorithm")
	// output file, default alg_result.csv
	outputPtr := flag.String("output", "example/output.csv", "output of this algorithm")
	flag.Parse()
	klog.InitFlags(nil)

	inputArray, err := parseInput(*inputPtr)
	exitWithError(err)

	outputArray, err := startSimulation(*algPtr, inputArray)
	exitWithError(err)

	err = parseResult(*outputPtr, outputArray)
	exitWithError(err)
}

func exitWithError(err error) {
	if err != nil {
		klog.Errorf("%v\n", err)
		os.Exit(1)
	}
}

// every row of the input file will be parsed to one instance of inputData
type inputData struct {
	// input id of the row
	name string
	// parse zone info of the input file into zone data structure
	zones []modeling.Zone
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
	result modeling.SimulationResult
}

// paserInput parses input csv file to instances of inputData and returns a
// slice of them
func parseInput(file string) ([]inputData, error) {
	inputFile, err := os.Open(filepath.Join("", filepath.Clean(file)))
	if err != nil {
		return nil, err
	}
	klog.Infof("Reading data from %v\n", file)
	reader := csv.NewReader(inputFile)
	reader.TrimLeadingSpace = true
	line, err := reader.Read()
	if err != nil {
		return nil, err
	}
	var zoneNames []string
	for _, name := range line[1:] {
		name = strings.TrimSpace(name)
		zoneNames = append(zoneNames, name)
	}
	var dataArray []inputData
	for {
		var data inputData
		var done bool
		data, done, err = readOneRow(zoneNames, reader)
		if done {
			break
		}
		if err != nil {
			klog.Infof("can't parse input data: %v, skip to next row\n", data.name)
			continue
		}
		dataArray = append(dataArray, data)
	}
	return dataArray, err
}

// parse one row of input file to one instance of inputData
func readOneRow(zoneNames []string, reader *csv.Reader) (inputData, bool, error) {
	rowCells, err := reader.Read()
	if err == io.EOF {
		return inputData{}, true, nil
	}
	if err != nil {
		return inputData{}, true, err
	}
	var rowData inputData
	rowData.name = rowCells[0]
	for index, data := range rowCells[1:] {
		nodestr := strings.Fields(data)
		// convert string to int. number of nodes in a zone
		numNode, err := strconv.Atoi(nodestr[0])
		if err != nil {
			return rowData, false, err
		}
		// convert string to int. number of endpoints in a zone
		numEndpoints, err := strconv.Atoi(nodestr[1])
		if err != nil {
			return rowData, false, err
		}
		rowData.zones = append(rowData.zones, modeling.Zone{
			Nodes:     numNode,
			Endpoints: numEndpoints,
			Name:      zoneNames[index],
		})
	}
	return rowData, false, nil
}

// startSimulation processes simulation on input data, produces instances of
// outputData structure and returns a slice of them
func startSimulation(algName string, inputArray []inputData) ([]outputData, error) {
	// create algrithm based on the algorithm name delivered by the flag
	alg := modeling.NewAlgorithm(algName)
	// create simulation model, currently do calculation based on probability
	// rather than real simulation.
	model, err := modeling.NewModel(alg, modeling.TheoreticalSimulator{})
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

// parseResult parses outputData to evaluation metrics and writes back to a
// result file
func parseResult(file string, outputArray []outputData) (err error) {
	outputFile, err := os.Create(file)
	if err != nil {
		return err
	}
	defer func() {
		cerr := outputFile.Close()
		if err == nil {
			err = cerr
		}
	}()

	klog.Infof("Writing output to file %v\n", file)
	writer := csv.NewWriter(outputFile)

	title := []string{"input name", "score", "in-zone-traffic score", "deviation score", "slice score", "max deviation", "mean deviation", "SD of deviation"}
	err = writer.Write(title)
	if err != nil {
		return err
	}

	for _, rowData := range outputArray {
		// use in zone traffic percentage to be in zone traffic score
		inZoneTrafficScore := rowData.result.InZoneTraffic * 100
		// use mean deviation to calcualte deviation score
		deviationScore := 100.0 - rowData.result.MeanDeviation*100
		// use number of EndpointSlices deviation to calculate sliceScore
		numberOfOriginalSlices := math.Ceil(float64(rowData.endpoints) / endpointsPerSlice)
		sliceScore := (numberOfOriginalSlices / float64(rowData.endpointSlices)) * 100
		// calculate total score based on two scores above
		totalScore := inZoneTrafficScoreWeight*inZoneTrafficScore + deviationScoreWeight*deviationScore + sliceScoreWeight*sliceScore

		data := []string{rowData.name}
		data = append(data, strconv.FormatFloat(totalScore, 'f', 4, 64))
		data = append(data, strconv.FormatFloat(inZoneTrafficScore, 'f', 4, 64))
		data = append(data, strconv.FormatFloat(deviationScore, 'f', 4, 64))
		data = append(data, strconv.FormatFloat(sliceScore, 'f', 4, 64))
		data = append(data, strconv.FormatFloat(rowData.result.MaxDeviation*100, 'f', 4, 64)+"%")
		data = append(data, strconv.FormatFloat(rowData.result.MeanDeviation*100, 'f', 4, 64)+"%")
		data = append(data, strconv.FormatFloat(rowData.result.DeviationSD, 'f', 4, 64))

		err = writer.Write(data)
		if err != nil {
			return err
		}
	}
	writer.Flush()
	err = writer.Error()
	return err
}
