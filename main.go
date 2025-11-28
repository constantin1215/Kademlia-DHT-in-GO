package main

import (
	"log"
	ks "peer/kademlia/service"
)

type Config struct {
	id           string
	ip           string
	port         int32
	k            int
	routingTable map[uint16][]*ks.NodeInfo
}

var config = Config{
	id:           "",
	ip:           getIP(),
	port:         7777,
	k:            4,
	routingTable: nil,
}

func main() {
	config.id = calculateNodeId()
	config.routingTable = initRoutingTable()
	log.Printf("Kademlia node(%s, %s, %d)", config.id, config.ip, config.port)
	go joinNetwork()
	startService()
}
