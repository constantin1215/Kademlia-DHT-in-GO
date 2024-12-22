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
	routingTable = initRoutingTable()
	data         = make(map[int32]int32)
)

const (
	k = 4
)

type server struct {
	ks.UnimplementedKademliaServiceServer
}

func (s *server) PING(_ context.Context, _ *ks.PingCheck) (*ks.NodeInfo, error) {
	return &ks.NodeInfo{Ip: ip, Port: int32(port), Id: id}, nil
}

func (s *server) STORE(_ context.Context, req *ks.StoreRequest) (*ks.StoreResult, error) {
	data[req.Key] = req.Value
	return &ks.StoreResult{Result: true, Message: "Data stored successfully!"}, nil
}

func (s *server) FIND_NODE(nodeId *ks.NodeID, stream grpc.ServerStreamingServer[ks.NodeInfo]) error {
	distance, err := calculateDistance(nodeId.Id, id)
	if err != nil {
		fmt.Println("Error calculating distance, invalid character detected in node ID.")
		return err
	}

	bucket := int8(math.Log2(float64(distance)) + 1)
	needed := k
	index := 0
	initialBucket := bucket
	bucketLen := len(routingTable[bucket])
	decrease := true

	for needed != 0 {
		if index < bucketLen {
			err := stream.Send(&routingTable[bucket][index])
			if err != nil {
				return err
			}
			index++
			needed--
		} else {
			if decrease { // initially decrease bucket from starting point
				bucket--
			} else {
				bucket++
			}

			if bucket == 0 { // if reached the bottom switch direction
				decrease = false
				bucket = initialBucket + 1
			}

			if bucket == 65 { // if out of bounds terminate
				return nil
			}

			index = 0
			bucketLen = len(routingTable[bucket])
		}
	}

	return nil
}

func (s *server) FIND_VALUE(_ context.Context, keyRequest *ks.Key) (*ks.Value, error) {
	return &ks.Value{Value: data[keyRequest.Key]}, nil
}
