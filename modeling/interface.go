package modeling

type RoutingAlgorithm interface {
	// This interface is to create endpointslices based on the current zones and the rouing algorithm
	//		Input: zones that involved in the routing
	//		Output: endpointslices that created based on the routing rules
	CreatingSlices([]Zone) ([]Endpointslice, error)
}

type TrafficSimulator interface {
	// This interface is to simulate the traffic among the zones
	//		Input: zones and endpointslices
	//		Output: Simulation results
	Simulate([]Zone, []Endpointslice) (Stat, error)
}
