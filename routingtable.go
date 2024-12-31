package main

import (
	"fmt"
	"log"
	ks "peer/kademlia/service"
)

var (
	nodePresence = make(map[string]bool)
)

func initRoutingTable() map[uint16][]*ks.NodeInfo {
	initial := make(map[uint16][]*ks.NodeInfo, 65)

	initial[0] = make([]*ks.NodeInfo, 0, 1)
	initial[0] = append(initial[0], &ks.NodeInfo{Ip: ip, Id: id, Port: port})
	for i := uint16(1); i <= 256; i++ {
		initial[i] = make([]*ks.NodeInfo, 0, k)
	}

	return initial
}

func updateRoutingTable(bucket uint16, newInfo *ks.NodeInfo) {
	insertedNew := false
	quickAccessKey := newInfo.Id
	if len(routingTable[bucket]) == k {
		log.Println("Bucket full. Checking for node to evict")
		for i, node := range routingTable[bucket] {
			client, conn, err := createClient(node.Ip)
			if err != nil {
				return
			}

			_, errPing := clientPerformPING(client)
			if errPing != nil {
				log.Println("Removing unresponsive node and adding it at the end of the bucket")
				routingTable[bucket][i] = routingTable[bucket][len(routingTable[bucket])-1]
				log.Printf("Evicted %v", routingTable[bucket][i])
				routingTable[bucket][len(routingTable[bucket])-1] = newInfo
				nodePresence[quickAccessKey] = true
				_ = conn.Close()
				insertedNew = true
				break
			}

			_ = conn.Close()
		}

	} else {
		_, ok := nodePresence[quickAccessKey]
		if !ok {
			routingTable[bucket] = append(routingTable[bucket], newInfo)
			nodePresence[quickAccessKey] = true
			insertedNew = true
		}
	}

	if !insertedNew {
		log.Println("New node could not be inserted in bucket")
	} else {
		log.Printf("Added new node in routing table. Bucket: %d Node: %v", bucket, newInfo)
	}
}

func gatherClosestNodes(bucket uint16, requesterId string, targetId string) []*ks.NodeInfoLookup {
	needed := k
	index := 0
	initialBucket := bucket
	bucketLen := len(routingTable[bucket])
	decrease := true
	nodes := make([]*ks.NodeInfoLookup, 0, k)

	for needed != 0 {
		if index < bucketLen {
			if routingTable[bucket][index].Id != requesterId { // skip the requesting node
				log.Printf("Checking bucket %d index %d node %v", bucket, index, routingTable[bucket][index])
				distanceToTarget, err := calculateDistance(targetId, routingTable[bucket][index].Id)
				if err == nil {
					nodes = append(nodes, &ks.NodeInfoLookup{NodeDetails: routingTable[bucket][index], DistanceToTarget: uint32(distanceToTarget)})
					needed--
				} else {
					log.Printf("Error calculating distance: %v", err)
				}
			}

			index++
		} else {
			if (bucket == 0 && initialBucket == 256) || (bucket == 256 && initialBucket != 256) { // already checked all buckets
				break
			}

			if bucket == 0 { // if reached the bottom switch direction
				decrease = false
				bucket = initialBucket
			}

			if decrease { // initially decrease bucket from starting point
				bucket--
			} else {
				bucket++
			}

			index = 0
			bucketLen = len(routingTable[bucket])
		}
	}

	return nodes
}

func findClosestNodes(requester *ks.NodeInfo, target string) ([]*ks.NodeInfoLookup, error) {
	bucket, err := calculateDistance(target, id)
	if err != nil {
		fmt.Println("Error calculating distance, invalid character detected in node ID.")
		return nil, err
	}

	var gatheredNodes []*ks.NodeInfoLookup

	if requester != nil {
		gatheredNodes = gatherClosestNodes(bucket, requester.Id, target)
	} else {
		gatheredNodes = gatherClosestNodes(bucket, "", target)
	}
	log.Printf("Gathered nodes %v", gatheredNodes)
	return gatheredNodes, nil
}
