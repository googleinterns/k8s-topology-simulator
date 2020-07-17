package modeling

import "errors"

// MockAlg1 is used for test, return single endpointslicegroup with size > 100
type MockAlg1 struct {
}

// CreateSlices return slicegroup size = 200
func (alg *MockAlg1) CreateSlices(zones zoneInfos) (map[string]EndpointSliceGroup, error) {
	if zones.zoneDetails == nil {
		return nil, errors.New("Can't create endpointslices with 0 number of zone")
	}
	return map[string]EndpointSliceGroup{"a": EndpointSliceGroup{
		Composition: map[string]int{"a": 200},
	}}, nil
}

// MockAlg2 is used for test, return single endpointslicegroup with size <= 100
type MockAlg2 struct {
}

// CreateSlices return slicegroup size = 50
func (alg *MockAlg2) CreateSlices(zones zoneInfos) (map[string]EndpointSliceGroup, error) {
	if zones.zoneDetails == nil {
		return nil, errors.New("Can't create endpointslices with 0 number of zone")
	}
	return map[string]EndpointSliceGroup{"a": EndpointSliceGroup{
		Composition: map[string]int{"a": 50},
	}}, nil
}
