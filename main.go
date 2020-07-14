package main

import "github.com/googleinterns/k8s-topology-simulator/modeling"

func main() {
	zoneA := modeling.Zone{Nodes: 30, Endpoints: 60, Name: "a"}
	zoneB := modeling.Zone{Nodes: 35, Endpoints: 70, Name: "b"}
	zoneC := modeling.Zone{Nodes: 50, Endpoints: 80, Name: "c"}
	zones := []modeling.Zone{zoneA, zoneB, zoneC}

	alg, _ := modeling.CreateAlg(0.4, 100)
	sim := modeling.CreateDefaultSim(100000)

	mo, _ := modeling.NewModel(zones, alg, sim)
	mo.StartSimulation()
	mo.PrintLastResult()
}
