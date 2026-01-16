package main

import (
	"log"
	"os"
	ks "peer/kademlia/service"
	"strconv"
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
	k:            3,
	routingTable: nil,
}

func main() {
	config.id = calculateNodeId()
	k, err := strconv.Atoi(os.Getenv("K"))
	if err != nil {
		log.Println("K is not an integer. Will use default K=3")
		k = 3
	}
	config.k = k
	config.routingTable = initRoutingTable()
	//loadData()
	log.Printf("Kademlia node(%s, %s, %d)", config.id, config.ip, config.port)
	go joinNetwork()
	startService()
}
