package main

import (
	"fmt"
	"log"
	ks "peer/kademlia/service"
	"sync"
)

var routingTableLock = sync.Mutex{}

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
	for _, node := range routingTable[bucket] {
		if node.Id == newInfo.Id {
			routingTableLock.Unlock()
			return
		}
	}

	if len(routingTable[bucket]) < config.k {
		routingTable[bucket] = append(routingTable[bucket], newInfo)
		routingTableLock.Unlock()
		return
	}

	bucketCopy := append([]*ks.NodeInfo(nil), routingTable[bucket]...)
	routingTableLock.Unlock()

	var evict *ks.NodeInfo
	for _, node := range bucketCopy {
		client, conn, err := createClient(node.Ip)
		if err != nil {
			evict = node
			break
		}

		_, errPing := Ping(client)
		_ = conn.Close()
		if errPing != nil {
			log.Printf("Unresponsive %v", node)
			evict = node
			break
		}
	}

	if evict == nil {
		return
	}

	routingTableLock.Lock()
	defer routingTableLock.Unlock()

	newBucket := make([]*ks.NodeInfo, 0, config.k)
	for _, node := range routingTable[bucket] {
		if node.Id != evict.Id {
			newBucket = append(newBucket, node)
		}
	}

	if len(newBucket) < config.k {
		newBucket = append(newBucket, newInfo)
	}

	routingTable[bucket] = newBucket
}

func extractBucketNodes(
	bucketContent []*ks.NodeInfo,
	requesterId string,
	targetId string,
) []*ks.NodeInfoLookup {

	nodes := make([]*ks.NodeInfoLookup, 0, config.k)

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

		nodes = append(nodes, &ks.NodeInfoLookup{
			NodeDetails:      candidateNode,
			DistanceToTarget: uint32(distanceToTarget),
		})
	}

	return nodes
}

func gatherClosestNodes(
	bucket uint16,
	requesterId string,
	targetId string,
	routingTable map[uint16][]*ks.NodeInfo,
) []*ks.NodeInfoLookup {
	log.Println("Gathering closest nodes")
	nodes := make([]*ks.NodeInfoLookup, 0, config.k)

	scanBucket := func(b uint16) bool {
		routingTableLock.Lock()
		bucketContent := append([]*ks.NodeInfo(nil), routingTable[b]...)
		routingTableLock.Unlock()

		if len(bucketContent) == 0 {
			return false
		}

		candidates := extractBucketNodes(bucketContent, requesterId, targetId)
		if len(candidates) == 0 {
			return false
		}

		remain := config.k - len(nodes)
		if remain <= 0 {
			return true
		}
		if remain > len(candidates) {
			remain = len(candidates)
		}

		nodes = append(nodes, candidates[:remain]...)
		return len(nodes) == config.k
	}

	for b := int(bucket); b >= 0; b-- {
		if scanBucket(uint16(b)) {
			return nodes
		}
	}

	for b := int(bucket) + 1; b <= 256; b++ {
		if scanBucket(uint16(b)) {
			return nodes
		}
	}

	return nodes
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
