package main

import (
	"log"
	"maps"
	ks "peer/kademlia/service"
	"slices"
	"sort"
	"sync"
	"sync/atomic"
)

const alpha = 3

func processFindNode(targetId string, magicCookie uint64, result map[string]*ks.NodeInfoLookup, mutexResult *sync.Mutex,
	waitGroup *sync.WaitGroup, node *ks.NodeInfoLookup) {
	defer waitGroup.Done()

	log.Printf("Processing %v", node)

	mutexResult.Lock()
	result[node.NodeDetails.Id] = node
	mutexResult.Unlock()

	client, conn, errClient := createClient(node.NodeDetails.Ip)
	if errClient != nil {
		return
	}

	lookup, errLookup := LookupNode(targetId, magicCookie, client)
	if errLookup != nil {
		return
	}

	errConn := conn.Close()
	if errConn != nil {
		return
	}

	log.Printf("Node %s returned the following nodes: %v", node.NodeDetails.Ip, lookup)
	for _, infoLookup := range lookup {
		mutexResult.Lock()
		result[infoLookup.NodeDetails.Id] = infoLookup
		mutexResult.Unlock()
	}
}

func processFindValue(targetKey string, magicCookie uint64, result map[string]*ks.NodeInfoLookup, value *int32, version *int32,
	mutexResult *sync.Mutex, waitGroup *sync.WaitGroup, cancel *atomic.Bool, node *ks.NodeInfoLookup) {
	defer waitGroup.Done()

	log.Printf("Processing %v", node)

	mutexResult.Lock()
	result[node.NodeDetails.Id] = node
	mutexResult.Unlock()

	client, conn, errClient := createClient(node.NodeDetails.Ip)
	if errClient != nil {
		return
	}

	lookup, errLookup := LookupValue(targetKey, magicCookie, client)
	if errLookup != nil {
		return
	}
	errConn := conn.Close()
	if errConn != nil {
		return
	}

	if lookup.Value != nil && lookup.Version != nil {
		log.Printf("Found value %v with version %v", lookup.Value, lookup.Version)
		value = lookup.Value
		version = lookup.Version
		cancel.Store(true)
		return
	}

	log.Printf("Node %s returned the following nodes: %v", node.NodeDetails.Ip, lookup)
	for _, infoLookup := range lookup.Nodes {
		mutexResult.Lock()
		result[infoLookup.NodeDetails.Id] = infoLookup
		mutexResult.Unlock()
	}

	if cancel.Load() {
		return
	}
}

func findNodePool(target string, magicCookie uint64, initNodes []*ks.NodeInfoLookup) []*ks.NodeInfoLookup {
	result := make(map[string]*ks.NodeInfoLookup)

	var wg sync.WaitGroup
	var mutexResult sync.Mutex

	bestDistance := 256
	for _, node := range initNodes {
		result[node.NodeDetails.Id] = node
		if int(node.DistanceToTarget) < bestDistance {
			bestDistance = int(node.DistanceToTarget)
		}
	}

	nodesToProcess := initNodes

	for {
		j := 0
		roundResult := make(map[string]*ks.NodeInfoLookup)
		for {
			for i := 0; i < alpha && j < len(nodesToProcess); i++ {
				wg.Add(1)
				go processFindNode(target, magicCookie, roundResult, &mutexResult, &wg, nodesToProcess[j])
				j++
			}
			wg.Wait()
			if j >= len(nodesToProcess) {
				break
			}
		}
		bestDistRound := 256
		for _, node := range roundResult {
			result[node.NodeDetails.Id] = node
			if int(node.DistanceToTarget) < bestDistRound {
				bestDistRound = int(node.DistanceToTarget)
			}
		}

		if bestDistRound >= bestDistance {
			break
		}

		bestDistance = bestDistRound
		nodesToProcess = nil
		for _, node := range roundResult {
			nodesToProcess = append(nodesToProcess, node)
		}
	}
	resultedNodes := slices.Collect(maps.Values(result))
	sort.Slice(resultedNodes, func(i, j int) bool {
		return resultedNodes[i].DistanceToTarget < resultedNodes[j].DistanceToTarget && resultedNodes[i].NodeDetails.Id < resultedNodes[j].NodeDetails.Id
	})

	return resultedNodes
}

func findValuePool(request *ks.LookupRequest, magicCookie uint64, initNodes []*ks.NodeInfoLookup) ([]*ks.NodeInfoLookup, int32, int32) {
	result := make(map[string]*ks.NodeInfoLookup)

	var value int32 = -1
	var version int32 = 0

	var wg sync.WaitGroup
	var mutexResult sync.Mutex

	cancel := atomic.Bool{}
	cancel.Store(false)

	bestDistance := 256
	for _, node := range initNodes {
		result[node.NodeDetails.Id] = node
		if int(node.DistanceToTarget) < bestDistance {
			bestDistance = int(node.DistanceToTarget)
		}
	}

	nodesToProcess := initNodes
	for {
		j := 0
		roundResult := make(map[string]*ks.NodeInfoLookup)
		for {
			for i := 0; i < alpha && j < len(nodesToProcess); i++ {
				wg.Add(1)
				go processFindValue(request.Target, magicCookie, roundResult, &value, &version, &mutexResult, &wg, &cancel, nodesToProcess[j])
				j++
			}
			wg.Wait()
			if j >= len(nodesToProcess) {
				break
			}
		}
		bestDistRound := 256
		for _, node := range roundResult {
			result[node.NodeDetails.Id] = node
			if int(node.DistanceToTarget) < bestDistRound {
				bestDistRound = int(node.DistanceToTarget)
			}
		}

		if bestDistRound >= bestDistance {
			break
		}

		bestDistance = bestDistRound
		nodesToProcess = nil
		for _, node := range roundResult {
			nodesToProcess = append(nodesToProcess, node)
		}
	}

	resultedNodes := slices.Collect(maps.Values(result))
	sort.Slice(resultedNodes, func(i, j int) bool {
		return resultedNodes[i].DistanceToTarget < resultedNodes[j].DistanceToTarget && resultedNodes[i].NodeDetails.Id < resultedNodes[j].NodeDetails.Id
	})

	return resultedNodes, value, version
}
