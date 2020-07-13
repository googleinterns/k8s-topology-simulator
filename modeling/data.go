package modeling

type Zone struct {
	Nodes     int
	Endpoints int
	Name      string
}

type Endpointslice struct {
	Label       string
	Composition map[string]int
	Weights     map[string]float64
}

type Traffic struct {
	OutgoingZone    string
	IncomingTraffic float64
	OutgoingTraffic map[string]float64
}

type Stat struct {
	InZoneTraffic float64
	Traffic       map[string]*Traffic
	Workload      map[string]float64
}

func (e Endpointslice) numberOfPods() int {
	total := 0
	for _, pods := range e.Composition {
		total += pods
	}
	return total
}
