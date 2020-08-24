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
	"encoding/csv"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/googleinterns/k8s-topology-simulator/modeling/types"
	"k8s.io/klog/v2"
)

// paserInput parses an input csv file to instances of inputData and puts them
// into a queue(channel)
func parseInput(file string) (<-chan inputData, error) {
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
	inputQueue := make(chan inputData)

	go func() {
		defer close(inputQueue)
		defer func() {
			cerr := inputFile.Close()
			if cerr != nil {
				klog.Errorf("close input file %s with an error %v", file, cerr)
			}
		}()

		for data, done, rerr := readOneRow(zoneNames, reader); !done; data, done, rerr = readOneRow(zoneNames, reader) {
			if rerr != nil {
				klog.Errorf("can't parse input data: %v, due to error: %v, skip to next row\n", data.name, err)
				continue
			}
			inputQueue <- data
		}
	}()

	return inputQueue, err
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
		nodeStr := strings.Fields(data)
		// convert string to int. number of nodes in a zone
		numNodes, err := strconv.Atoi(nodeStr[0])
		if err != nil {
			return rowData, false, err
		}
		// convert string to int. number of endpoints in a zone
		numEndpoints, err := strconv.Atoi(nodeStr[1])
		if err != nil {
			return rowData, false, err
		}
		rowData.zones = append(rowData.zones, types.Zone{
			Nodes:     numNodes,
			Endpoints: numEndpoints,
			Name:      zoneNames[index],
		})
	}
	return rowData, false, nil
}
