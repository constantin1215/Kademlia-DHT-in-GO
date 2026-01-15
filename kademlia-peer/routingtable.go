package main

import (
	"fmt"
	"log"
	ks "peer/kademlia/service"
	"sort"
	"sync"
)

var routingTableLock = sync.Mutex{}

var (
	nodePresence = make(map[string]bool)
)

func initRoutingTable() map[uint16][]*ks.NodeInfo {
	initial := make(map[uint16][]*ks.NodeInfo, 256)

	initial[0] = make([]*ks.NodeInfo, 0, 1)
	initial[0] = append(initial[0], &ks.NodeInfo{Ip: config.ip, Id: config.id, Port: config.port})
	for i := uint16(1); i <= 256; i++ {
		initial[i] = make([]*ks.NodeInfo, 0, config.k)
	}

	return initial
}

func updateRoutingTable(bucket uint16, newInfo *ks.NodeInfo, routingTable map[uint16][]*ks.NodeInfo) {
	routingTableLock.Lock()
	if len(routingTable[bucket]) != config.k {
		_, ok := nodePresence[newInfo.Id]
		if !ok {
			routingTable[bucket] = append(routingTable[bucket], newInfo)
			nodePresence[newInfo.Id] = true
		}
		routingTableLock.Unlock()
		return
	}

	log.Println("Bucket full. Checking for node to evict")
	alive := make([]*ks.NodeInfo, 0)
	for _, node := range routingTable[bucket] {
		client, conn, err := createClient(node.Ip)
		if err != nil {
			continue
		}

		_, errPing := Ping(client)
		_ = conn.Close()
		if errPing != nil {
			log.Printf("Unresponsive %v", node)
			continue
		}

		alive = append(alive, node)
	}

	alive = append(alive, newInfo)
	if len(alive) > 1 {
		sort.Slice(alive, func(i, j int) bool {
			return alive[i].Id < alive[j].Id
		})
	}
	if len(alive) > config.k {
		routingTable[bucket] = alive[:config.k]
		routingTableLock.Unlock()
		return
	}

	routingTable[bucket] = alive
	routingTableLock.Unlock()
}

func gatherClosestNodes(bucket uint16, requesterId string, targetId string, routingTable map[uint16][]*ks.NodeInfo) []*ks.NodeInfoLookup {
	log.Println("Gathering closest nodes")
	nodes := make([]*ks.NodeInfoLookup, 0, config.k)

	for nextBucket := int(bucket); nextBucket >= 0; nextBucket-- {
		routingTableLock.Lock()
		bucketContent, ok := routingTable[uint16(nextBucket)]
		if !ok || len(bucketContent) == 0 {
			routingTableLock.Unlock()
			continue
		}

		candidateNodes, updatedBucket := extractBucketNodes(nextBucket, bucketContent, requesterId, targetId)
		routingTable[uint16(nextBucket)] = updatedBucket
		routingTableLock.Unlock()
		if len(candidateNodes) == 0 {
			continue
		}

		remain := config.k - len(nodes)
		if remain <= 0 {
			continue
		}

		if remain > len(candidateNodes) {
			remain = len(candidateNodes)
		}

		nodes = append(nodes, candidateNodes[:remain]...)
		if len(nodes) == config.k {
			return nodes
		}
	}

	for nextBucket := int(bucket) + 1; nextBucket <= 256; nextBucket++ {
		routingTableLock.Lock()
		bucketContent, ok := routingTable[uint16(nextBucket)]
		if !ok || len(bucketContent) == 0 {
			routingTableLock.Unlock()
			continue
		}

		candidateNodes, updatedBucket := extractBucketNodes(nextBucket, bucketContent, requesterId, targetId)
		routingTable[uint16(nextBucket)] = updatedBucket
		routingTableLock.Unlock()
		if len(candidateNodes) == 0 {
			continue
		}

		remain := config.k - len(nodes)
		if remain <= 0 {
			continue
		}

		if remain > len(candidateNodes) {
			remain = len(candidateNodes)
		}

		nodes = append(nodes, candidateNodes[:remain]...)
		if len(nodes) == config.k {
			return nodes
		}
	}

	return nodes
}

func extractBucketNodes(bucket int, bucketContent []*ks.NodeInfo, requesterId string, targetId string) ([]*ks.NodeInfoLookup, []*ks.NodeInfo) {
	nodes := make([]*ks.NodeInfoLookup, 0, config.k)
	workingNodes := make([]*ks.NodeInfo, 0, config.k)
	log.Printf("Bucket %v content: %v", bucket, bucketContent)
	for _, candidateNode := range bucketContent {
		if candidateNode.Id == requesterId {
			continue
		}

		distanceToTarget, err := calculateDistance(targetId, candidateNode.Id)
		if err != nil {
			continue
		}

		client, conn, err := createClient(candidateNode.Ip)
		if err != nil {
			continue
		}

		_, errPing := Ping(client)
		_ = conn.Close()
		if errPing != nil {
			log.Printf("Unresponsive %v", candidateNode)
			continue
		}

		nodes = append(nodes, &ks.NodeInfoLookup{NodeDetails: candidateNode, DistanceToTarget: uint32(distanceToTarget)})
		workingNodes = append(workingNodes, candidateNode)
	}

	log.Printf("Working nodes: %v", workingNodes)
	log.Printf("New bucket %v content: %v", bucket, bucketContent)

	return nodes, workingNodes
}

func findClosestNodes(requester *ks.NodeInfo, target string, routingTable map[uint16][]*ks.NodeInfo) ([]*ks.NodeInfoLookup, error) {
	startingBucket, err := calculateDistance(target, config.id)
	if err != nil {
		fmt.Println("Error calculating distance, invalid character detected in node ID.")
		return nil, err
	}

	var gatheredNodes []*ks.NodeInfoLookup

	if requester != nil {
		gatheredNodes = gatherClosestNodes(startingBucket, requester.Id, target, routingTable)
	} else {
		gatheredNodes = gatherClosestNodes(startingBucket, "", target, routingTable)
	}

	if len(gatheredNodes) == 0 {
		joinNetwork()
	}

	log.Printf("Gathered nodes %v", gatheredNodes)
	return gatheredNodes, nil
}
