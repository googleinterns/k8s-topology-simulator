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

// parseInput parses input csv file to instances of inputData and returns a
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
