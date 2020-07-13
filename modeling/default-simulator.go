package modeling

import (
	"math/rand"
	"time"
)

type DefaultSimulator struct {
	simulationTimes uint64
}

func CreateDefaultSim(times uint64) DefaultSimulator {
	sim := DefaultSimulator{simulationTimes: times}
	return sim
}

func (sim DefaultSimulator) Simulate(zones []Zone, slices []Endpointslice) (Stat, error) {
	var totalNodes int
	var totalPods int
	totalWeights := make(map[string]int)
	zoneNodes := make(map[string]int)
	zonePods := make(map[string]int)
	zoneWeights := make(map[string]map[string]float64)

	for _, zone := range zones {
		totalWeight := 0.0
		totalNodes += zone.Nodes
		totalPods += zone.Endpoints
		zoneWeights[zone.Name] = make(map[string]float64)
		for _, slice := range slices {
			if val, ok := slice.Weights[zone.Name]; ok {
				tmp := float64(slice.numberOfPods()) * val
				totalWeight += tmp
				zoneWeights[zone.Name][slice.Label] = tmp
			}
		}
		totalWeights[zone.Name] = int(totalWeight)
		zoneNodes[zone.Name] = zone.Nodes
		zonePods[zone.Name] = zone.Endpoints
	}

	inZoneTraffic := 0
	detailTraffic := make(map[string]map[string]uint64)
	hitTimes := make(map[string]uint64)
	for i := uint64(0); i < sim.simulationTimes; i++ {
		incomingZone := simulationHelperInt(zoneNodes, totalNodes)
		hitSliceLabel := simulationHelperFloat(zoneWeights[incomingZone], totalWeights[incomingZone])
		hitSlice := 0
		for index, slice := range slices {
			if hitSliceLabel == slice.Label {
				hitSlice = index
			}
		}
		hitZone := simulationHelperInt(slices[hitSlice].Composition, slices[hitSlice].numberOfPods())

		if incomingZone == hitZone {
			inZoneTraffic++
		}
		hitTimes[hitZone]++
		if _, ok := detailTraffic[incomingZone]; !ok {
			detailTraffic[incomingZone] = make(map[string]uint64)
		}
		detailTraffic[incomingZone][hitZone]++
	}

	var stat Stat
	stat.InZoneTraffic = float64(inZoneTraffic) / float64(sim.simulationTimes)
	stat.Traffic = make(map[string]*Traffic)
	stat.Workload = make(map[string]float64)
	for zone, traffic := range detailTraffic {
		stat.Traffic[zone] = new(Traffic)
		stat.Traffic[zone].OutgoingZone = zone
		stat.Traffic[zone].IncomingTraffic = float64(hitTimes[zone]) / float64(sim.simulationTimes)
		stat.Workload[zone] = stat.Traffic[zone].IncomingTraffic / float64(zonePods[zone]) * float64(totalPods)
		totalOutgoing := uint64(0)
		for _, times := range traffic {
			totalOutgoing += times
		}
		stat.Traffic[zone].OutgoingTraffic = make(map[string]float64)
		for hittenZone, times := range traffic {
			stat.Traffic[zone].OutgoingTraffic[hittenZone] = float64(times) / float64(totalOutgoing)
		}
	}
	return stat, nil
}

func simulationHelperInt(probs map[string]int, total int) string {
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	tmp := random.Intn(total)
	acc := 0
	for key, val := range probs {
		upper := acc + val
		if tmp >= acc && tmp < upper {
			return key
		}
		acc = upper
	}
	return ""
}

func simulationHelperFloat(probs map[string]float64, total int) string {
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	tmp := random.Intn(total)
	acc := 0.
	for key, val := range probs {
		upper := acc + val
		if float64(tmp) >= acc && float64(tmp) < upper {
			return key
		}
		acc = upper
	}
	return ""
}
