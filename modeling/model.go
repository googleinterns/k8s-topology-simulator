package modeling

import (
	"errors"
	"fmt"
)

type Model struct {
	zones      []Zone
	slices     []Endpointslice
	lastResult Stat
	alg        RoutingAlgorithm
	simulator  TrafficSimulator
}

func NewModel(zones []Zone, alg RoutingAlgorithm, sim TrafficSimulator) (*Model, error) {
	model := &Model{
		zones:     zones,
		alg:       alg,
		simulator: sim,
	}
	if slices, err := model.alg.CreatingSlices(model.zones); err != nil {
		panic(err)
	} else {
		model.slices = slices
	}
	return model, nil
}

func (m *Model) StartSimulation() error {
	if stat, err := m.simulator.Simulate(m.zones, m.slices); err != nil {
		panic(err)
	} else {
		m.lastResult = stat
	}
	return nil
}

func (m *Model) UpdateAlgorithm(alg RoutingAlgorithm) error {
	if alg == nil {
		return errors.New("Empty algorithm")
	}
	m.alg = alg
	if slices, err := m.alg.CreatingSlices(m.zones); err != nil {
		panic(err)
	} else {
		m.slices = slices
	}
	return nil
}

func (m *Model) PrintLastResult() error {
	var totalPods int
	for _, zone := range m.zones {
		totalPods += zone.Endpoints
	}
	fmt.Printf("%% in-zone traffic \t %.2f%%\n", m.lastResult.InZoneTraffic*100)
	fmt.Printf("# of endpoint slices\t %d\n", len(m.slices))
	fmt.Printf("# of endpoints\t %d\n", totalPods)
	fmt.Println("----------------------------------------------")

	for zone, traffic := range m.lastResult.Traffic {
		fmt.Printf("Total to %s \t %.f%% \n", zone, traffic.IncomingTraffic*100)
	}
	fmt.Println("----------------------------------------------")

	for zone, traffic := range m.lastResult.Traffic {
		fmt.Printf("From %s : \n", zone)
		for z, t := range traffic.OutgoingTraffic {
			fmt.Printf("\t to %s : %.2f%% \n", z, t*100)
		}
	}
	fmt.Println("----------------------------------------------")

	for zone, workload := range m.lastResult.Workload {
		fmt.Printf("Workload for %s : \t %.2f%% \n", zone, workload*100)
	}
	return nil
}

func (m *Model) GetZones() ([]Zone, error) {
	if len(m.zones) == 0 {
		// Suppose to be initialized outside, if empty panic!
		panic(errors.New("Can't get empty zones"))
	}
	return m.zones, nil
}

func (m *Model) GetEndpointslices() ([]Endpointslice, error) {
	if len(m.slices) == 0 {
		return nil, errors.New("Can't get empty slices")
	}
	return m.slices, nil
}

func (m *Model) GetLastResult() (Stat, error) {
	if m.lastResult.Traffic == nil {
		return m.lastResult, errors.New("No stats yet, run simulation first")
	}
	return m.lastResult, nil
}
