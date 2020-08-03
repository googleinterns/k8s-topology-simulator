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
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/googleinterns/k8s-topology-simulator/modeling"
)

func main() {
	// algorithm name, default shared global
	algPtr := flag.String("alg", "SharedGlobalAlgorithm", "routing algorithm")
	// input file
	inputPtr := flag.String("input", "example/input.csv", "inputs to use for this algorithm")
	// output file, default alg_result.csv
	outputPtr := flag.String("output", "example/output.csv", "output of this algorithm")
	flag.Parse()

	var errList []<-chan error

	zoneNames, inputQueue, errC, err := parseInput(*inputPtr)
	errorHandler(err)
	errList = append(errList, errC)

	outputQueue, errC, err := startSimulation(*algPtr, inputQueue)
	errorHandler(err)
	errList = append(errList, errC)

	errC, err = parseResult(*outputPtr, zoneNames, outputQueue)
	errorHandler(err)
	errList = append(errList, errC)

	// wait for goroutines to return
	err = waitForSimulation(errList...)
	errorHandler(err)
}

func errorHandler(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
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
	// simulation result of that piece of input data
	result modeling.SimulationResult
}

// paserInput parses input csv file to instances of inputData and puts them into
// a queue
func parseInput(file string) ([]string, <-chan inputData, <-chan error, error) {
	inputFile, err := os.Open(filepath.Join("", filepath.Clean(file)))
	if err != nil {
		return nil, nil, nil, err
	}
	fmt.Printf("Reading data from %v\n", file)
	reader := csv.NewReader(inputFile)
	reader.TrimLeadingSpace = true
	line, err := reader.Read()
	if err != nil {
		return nil, nil, nil, err
	}
	var zoneNames []string
	for _, name := range line[1:] {
		name = strings.TrimSpace(name)
		zoneNames = append(zoneNames, name)
	}
	dataC := make(chan inputData)
	errC := make(chan error, 1)

	go func() {
		defer close(dataC)
		defer close(errC)
		for {
			// read one row
			rowCells, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				errC <- err
				return
			}
			if len(rowCells) != len(zoneNames)+1 {
				fmt.Printf("[WARNING] unmatched number of fields, skip data %v\n", rowCells)
				continue
			}
			var rowData inputData
			rowData.name = rowCells[0]
			for index, data := range rowCells[1:] {
				nodestr := strings.Fields(data)
				// convert string to int. number of nodes in a zone
				numNode, err := strconv.Atoi(nodestr[0])
				if err != nil {
					errC <- err
					return
				}
				// convert string to int. number of endpoints in a zone
				numEndpoints, err := strconv.Atoi(nodestr[1])
				if err != nil {
					errC <- err
					return
				}
				rowData.zones = append(rowData.zones, modeling.Zone{
					Nodes:     numNode,
					Endpoints: numEndpoints,
					Name:      zoneNames[index],
				})
			}
			dataC <- rowData
		}
	}()

	return zoneNames, dataC, errC, nil
}

// startSimulation processes simulation on input data, produces instances of
// outputData structure and puts them in a queue
func startSimulation(algName string, inputQueue <-chan inputData) (<-chan outputData, <-chan error, error) {
	// create algrithm based on the algorithm name delivered by the flag
	alg := modeling.NewAlgorithm(algName)
	// create simulation model, currently do calculation based on probability
	// rather than real simulation.
	model, err := modeling.NewModel(alg, modeling.TheoreticalSimulator{})
	if err != nil {
		return nil, nil, err
	}

	resultC := make(chan outputData)
	errC := make(chan error, 1)

	go func() {
		defer close(resultC)
		defer close(errC)
		for {
			rowData, more := <-inputQueue
			if !more {
				break
			}
			err := model.UpdateRegion(rowData.zones)
			if err != nil {
				errC <- fmt.Errorf("input id : %s, %v", rowData.name, err)
				return
			}
			simRes, err := model.StartSimulation()
			if err != nil {
				errC <- fmt.Errorf("input id : %s, %v", rowData.name, err)
				return
			}
			var result outputData
			result.name = rowData.name
			result.result = simRes
			resultC <- result
		}
	}()

	return resultC, errC, nil
}

// parseResult parses outputData to evaluation metrics and writes back to a
// result file
func parseResult(file string, zoneNames []string, outputQueue <-chan outputData) (<-chan error, error) {
	outputFile, err := os.Create(file)
	if err != nil {
		return nil, err
	}
	defer func() {
		cerr := outputFile.Close()
		if err == nil {
			err = cerr
		}
	}()
	fmt.Printf("Creating output to file %v\n", file)
	writer := csv.NewWriter(outputFile)

	title := []string{"input id", "score", "in-zone-traffic score", "deviation score", "max deviation", "mean deviation", "SD of deviation"}
	err = writer.Write(title)
	if err != nil {
		return nil, err
	}
	errC := make(chan error, 1)
	defer close(errC)
	for {
		rowData, more := <-outputQueue
		if !more {
			break
		}

		// use in zone traffic percentage to be in zone traffic score
		inZoneTrafficScore := rowData.result.InZoneTraffic * 100
		// use mean deviation to calcualte deviation score
		deviationScore := 100.0 - rowData.result.MeanDeviation*100
		// calculate total score based on two scores above
		totalScore := 0.6*inZoneTrafficScore + 0.4*deviationScore

		data := []string{rowData.name}
		data = append(data, strconv.FormatFloat(totalScore, 'f', 4, 64))
		data = append(data, strconv.FormatFloat(inZoneTrafficScore, 'f', 4, 64))
		data = append(data, strconv.FormatFloat(deviationScore, 'f', 4, 64))
		data = append(data, strconv.FormatFloat(rowData.result.MaxDeviation*100, 'f', 4, 64)+"%")
		data = append(data, strconv.FormatFloat(rowData.result.MeanDeviation*100, 'f', 4, 64)+"%")
		data = append(data, strconv.FormatFloat(rowData.result.DeviationSD, 'f', 4, 64))

		err = writer.Write(data)
		if err != nil {
			return errC, err
		}
	}
	writer.Flush()
	err = writer.Error()
	if err != nil {
		return errC, err
	}
	return errC, nil
}

// waitForSimulation waits for above routings return and handle their errors
func waitForSimulation(errs ...<-chan error) error {
	errC := mergeErrors(errs...)
	for err := range errC {
		if err != nil {
			return err
		}
	}
	return nil
}

// mergeErrors handles errors in the goroutines
func mergeErrors(errs ...<-chan error) <-chan error {
	var wg sync.WaitGroup
	outC := make(chan error, len(errs))
	output := func(c <-chan error) {
		for err := range c {
			outC <- err
		}
		wg.Done()
	}
	wg.Add(len(errs))

	for _, errC := range errs {
		go output(errC)
	}

	go func() {
		wg.Wait()
		close(outC)
	}()
	return outC
}
