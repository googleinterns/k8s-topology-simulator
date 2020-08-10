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
	"flag"
	"os"

	"github.com/googleinterns/k8s-topology-simulator/process"
	"k8s.io/klog/v2"
)

func main() {
	// algorithm name, default shared global
	algPtr := flag.String("alg", "SharedGlobalAlgorithm", "routing algorithm")
	// input file
	inputPtr := flag.String("input", "example/input.csv", "inputs to use for this algorithm")
	// output file, default alg_result.csv
	outputPtr := flag.String("output", "example/output.csv", "output of this algorithm")
	flag.Parse()
	klog.InitFlags(nil)

	err := process.StartProcessing(*inputPtr, *outputPtr, *algPtr)
	exitWithError(err)
}

func exitWithError(err error) {
	if err != nil {
		klog.Errorf("%v\n", err)
		os.Exit(1)
	}
}
