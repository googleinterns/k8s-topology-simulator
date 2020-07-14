package modeling

import (
	"math/rand"
	"time"
)

type DefaultSimulator struct {
	// Times to run the simulation, larger more accurate but slower
	simulationTimes uint64
}

func CreateDefaultSim(times uint64) DefaultSimulator {
	sim := DefaultSimulator{simulationTimes: times}
	return sim
}

func (sim DefaultSimulator) Simulate(zones []Zone, slices []Endpointslice) (Stat, error) {
	// Total number of endpoints and nodes of all zones
	var totalNodes int
	var totalPods int
	// Weighted number of endpoints that in a zone
	totalWeights := make(map[string]int)
	// Restruct the slice structure to map structure to use the helper function, zonename - number of pods/nodes
	zoneNodes := make(map[string]int)
	zonePods := make(map[string]int)
	// Composition of the weighted number of endpoints in a zone
	zoneWeights := make(map[string]map[string]float64)

	// Calculate the weighted number of pods and composition of different zones
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

	// Total ratio of traffic stays in the same zone
	inZoneTraffic := 0
	// Detailed traffic information of every single zone
	detailTraffic := make(map[string]map[string]uint64)
	// How many requests that hit a zone specifically
	hitTimes := make(map[string]uint64)
	for i := uint64(0); i < sim.simulationTimes; i++ {
		// Simulate the a random request from a zone, incomingZone is the name of the zone making this request
		incomingZone := simulationHelperInt(zoneNodes, totalNodes)
		// Simulate the slice that request is going to, hitSliceLabel is the lable of the hit slice
		hitSliceLabel := simulationHelperFloat(zoneWeights[incomingZone], totalWeights[incomingZone])
		hitSlice := 0
		// Get that slice based on the lable
		for index, slice := range slices {
			if hitSliceLabel == slice.Label {
				hitSlice = index
			}
		}
		// Simulate the zone that request is going to, hitZone is the name of the zone receiving this request
		hitZone := simulationHelperInt(slices[hitSlice].Composition, slices[hitSlice].numberOfPods())

		// If origin and destination is the same, this is an inzone traffic
		if incomingZone == hitZone {
			inZoneTraffic++
		}
		hitTimes[hitZone]++
		if _, ok := detailTraffic[incomingZone]; !ok {
			detailTraffic[incomingZone] = make(map[string]uint64)
		}
		// This contributes to the distribution part of the result
		detailTraffic[incomingZone][hitZone]++
	}

	var stat Stat
	stat.InZoneTraffic = float64(inZoneTraffic) / float64(sim.simulationTimes)
	stat.Traffic = make(map[string]*Traffic)
	stat.Workload = make(map[string]float64)
	// Calculte the metrics of the result, should be quite straightforward
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

// Following two functions are helper to simulate a random request and return the name/label of the 'hit' zone/endpointslice
//		probs: distributions
//		total: sum of the distributions
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
