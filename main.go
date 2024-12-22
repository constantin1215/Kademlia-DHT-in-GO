package main

import (
	"fmt"
	"google.golang.org/grpc"
	"log"
	"net"
	ks "peer/kademlia/service"
)

func main() {
	networkPeers, _ := scanNetwork()
	for i := range networkPeers {
		fmt.Println(networkPeers[i])
	}

	startServer()
}

func startServer() {
	lis, err := net.Listen("tcp4", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	ks.RegisterKademliaServiceServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
