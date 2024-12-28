package main

import (
	"log"
)

func main() {
	log.Printf("Kademlia node(%s, %s, %d)", id, ip, port)
	go joinNetwork()
	startService()
}
