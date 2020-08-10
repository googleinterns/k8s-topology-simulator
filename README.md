# Kubernetes Topology Simulator

**This is not an officially supported Google product.**

This will be a key part in the planning and design of topology aware routing for
Kubernetes. This specific project will include:

* Building a tool that can be used to test the effectiveness of different
  topology aware routing algorithms.
* Tweaking algorithms that have already been proposed or propose new ones.
* Writing a report that summarizes the different approaches that can be used for
  topology aware routing along with a recommendation.

## Usage
go run main.go -input=inputFile -output=outputFile -alg=algorithm

example of intput file (csv): each zone with number of nodes first, number of endpoints next
```
input name, zone1, zone2, zone3  
perfect input, 10 10, 10 10, 20 20
```

## Interfaces
1. Implement algorithms comply with the `RoutingAlgorithm` interface.
```
type RoutingAlgorithm struct {
    // CreateSliceGroups translates regionInfo into EndpointSliceGroups
    CreateSliceGroups(data.RegionInfo) (map[string]data.EndpointSliceGroup, error)
}
```

2. Add an entry of the algorithm to `NewAlgorithm(name string) RoutingAlgorithm` introduced in algorithms.go
