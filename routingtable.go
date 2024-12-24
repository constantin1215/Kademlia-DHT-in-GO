package main

import (
	"log"
	ks "peer/kademlia/service"
	"strconv"
)

func initRoutingTable() map[uint8][]*ks.NodeInfo {
	initial := make(map[uint8][]*ks.NodeInfo, 65)

	for i := uint16(0); i < 256; i++ {
		initial[uint8(i)] = make([]*ks.NodeInfo, 0, 10)
	}

	return initial
}

func updateRoutingTable(bucket uint8, newInfo *ks.NodeInfo) {
	_, ok := quickAccessRoutingTable[strconv.Itoa(int(bucket))+"-"+newInfo.Id] // optimize later
	if !ok {
		routingTable[bucket] = append(routingTable[bucket], newInfo)
		quickAccessRoutingTable[strconv.Itoa(int(bucket))+"-"+newInfo.Id] = newInfo
		log.Printf("Added new node in routing table. Bucket: %d Bucket content: %v", bucket, routingTable[bucket])
	} else {
		// check if bucket is full
	}
}
