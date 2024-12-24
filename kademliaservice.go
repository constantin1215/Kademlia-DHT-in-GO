package main

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"net"
	ks "peer/kademlia/service"
)

var (
	id                      = calculateNodeId()
	routingTable            = initRoutingTable()
	quickAccessRoutingTable = make(map[string]*ks.NodeInfo)
	data                    = make(map[int32]int32)
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

func (s *server) FIND_NODE(request *ks.LookupRequest, stream grpc.ServerStreamingServer[ks.NodeInfo]) error {
	bucket, err := calculateDistance(request.TargetNodeId, id)
	if err != nil {
		fmt.Println("Error calculating distance, invalid character detected in node ID.")
		return err
	}

	log.Printf("Looking in bucket %d", bucket)

	needed := k
	index := 0
	initialBucket := bucket
	bucketLen := len(routingTable[bucket])
	decrease := true
	reachedZero := 0

	for needed != 0 {
		if reachedZero == 2 {
			break
		}
		if index < bucketLen {
			log.Printf("Checking bucket %d index %d node %v", bucket, index, routingTable[bucket][index])
			err := stream.Send(routingTable[bucket][index])
			if err != nil {
				break
			}
			log.Printf("Send %v", routingTable[bucket][index])
			index++
			needed--
		} else {
			if bucket == 0 && initialBucket == 255 { // already checked all buckets
				break
			}

			if bucket == 0 { // if reached the bottom switch direction
				reachedZero++
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

	log.Printf("Adding %v to routing table", request.Requester)
	updateRoutingTable(initialBucket, request.Requester)

	return nil
}

func (s *server) FIND_VALUE(_ context.Context, keyRequest *ks.Key) (*ks.Value, error) {
	return &ks.Value{Value: data[keyRequest.Key]}, nil
}

func startService() {
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
