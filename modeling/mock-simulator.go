package modeling

import "errors"

// MockSimulator is used for test : verify inputs and return dummy stat
type MockSimulator struct {
}

// Simulate return dummy stat
func (sim *MockSimulator) Simulate(zones zoneInfos, endpointSlices map[string]EndpointSliceGroup) (Stat, error) {
	if len(zones.zoneDetails) == 0 || len(endpointSlices) == 0 {
		return Stat{}, errors.New("Can't evaluate probability based on empty zones or endpointslices")
	}
	stat := Stat{InZoneTraffic: 1.0}
	return stat, nil
}
