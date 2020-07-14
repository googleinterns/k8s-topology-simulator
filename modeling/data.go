package modeling

type Zone struct {
	// Number of nodes of this zone
	Nodes int
	// Number of endpoitns of this zone
	Endpoints int
	// Name of this zone
	Name string
}

type Endpointslice struct {
	// Lable of this endpointslice
	Label string
	// Contribution of endpoints from different zones
	Composition map[string]int
	// Weights that this endpointslice for different zones while routing
	Weights map[string]float64
}

type Traffic struct {
	// Name of a specific zone
	OutgoingZone string
	// Traffic that the same zone received
	IncomingTraffic float64
	// Outgoing traffic distribution of the same zone
	OutgoingTraffic map[string]float64
}

type Stat struct {
	// Total ratio of traffic that stays in the same zone
	InZoneTraffic float64
	// Traffic details for different zones
	Traffic map[string]*Traffic
	// Workload balance for different zones -- ratio of incoming traffic / ratio of capacity
	Workload map[string]float64
}

// Helper function to calculte number of endpoints of a specific endpointslice
func (e Endpointslice) numberOfPods() int {
	total := 0
	for _, pods := range e.Composition {
		total += pods
	}
	return total
}
