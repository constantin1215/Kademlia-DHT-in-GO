package main

import (
	ks "peer/kademlia/service"
)

func initRoutingTable() map[int8][]ks.NodeInfo {
	initial := make(map[int8][]ks.NodeInfo, 65)

	for i := int8(0); i < 65; i++ {
		initial[i] = make([]ks.NodeInfo, 0, 10)
	}

	return initial
}
