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

import "github.com/googleinterns/k8s-topology-simulator/modeling"

func main() {
	zoneA := modeling.Zone{Nodes: 30, Endpoints: 60, Name: "a"}
	zoneB := modeling.Zone{Nodes: 35, Endpoints: 70, Name: "b"}
	zoneC := modeling.Zone{Nodes: 50, Endpoints: 80, Name: "c"}
	zones := []modeling.Zone{zoneA, zoneB, zoneC}

	alg, err := modeling.CreateAlg(0.4, 100)
	if err != nil {
		panic(err)
	}

	var sim modeling.TheoreticalSimulator

	// TODO: operations on metrics rather than printing only
	model, err := modeling.NewModel(zones, alg, sim)
	if err != nil {
		panic(err)
	}

	model.StartSimulation()
	model.PrintLastResult()
}
