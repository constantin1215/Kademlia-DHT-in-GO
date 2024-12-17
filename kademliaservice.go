package main

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"math"
	ks "peer/kademlia/service"
)

var (
	id           = calculateNodeId()
	routingTable = make(map[int][]int)
)

type server struct {
	ks.UnimplementedKademliaServiceServer
}

func (s *server) PING(_ context.Context, _ *ks.PingCheck) (*ks.NodeInfo, error) {
	return &ks.NodeInfo{Ip: getIP(), Port: int32(*port), Id: id}, nil
}

func (s *server) FIND_NODE(nodeId *ks.NodeID, stream grpc.ServerStreamingServer[ks.NodeInfo]) error {
	distance, err := calculateDistance(nodeId.Id, id)

	fmt.Println(id)

	if err != nil {
		fmt.Println("Error calculating distance, invalid character detected in node ID.")
		return err
	}

	bucket := int8(math.Log2(float64(distance)) + 1)

	fmt.Printf("Distance %d\nBucket %d\n", distance, bucket)

	return nil
}
