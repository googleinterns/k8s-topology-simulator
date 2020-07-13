package modeling

type RoutingAlgorithm interface {
	CreatingSlices([]Zone) ([]Endpointslice, error)
}

type TrafficSimulator interface {
	Simulate([]Zone, []Endpointslice) (Stat, error)
}
