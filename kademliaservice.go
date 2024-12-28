package main

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"math/rand/v2"
	"net"
	ks "peer/kademlia/service"
	"sort"
	"strconv"
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

func (s *server) FIND_NODE(_ context.Context, request *ks.LookupRequest) (*ks.LookupResponse, error) {
	log.Printf("FIND_NODE: Request received: %v", request)

	bucket, err := calculateDistance(request.TargetNodeId, id)
	if err != nil {
		fmt.Println("Error calculating distance, invalid character detected in node ID.")
		return nil, err
	}

	var nodesQueue []*ks.NodeInfoLookup
	var result []*ks.NodeInfoLookup
	visited := make(map[string]bool)
	if request.Requester != nil {
		nodesQueue = gatherClosestNodes(bucket, request.Requester.Id, request.TargetNodeId)
	} else {
		nodesQueue = gatherClosestNodes(bucket, "", request.TargetNodeId)
	}
	log.Printf("Gathered nodes %v", nodesQueue)

	if request.MagicCookie == nil {
		magicCookie := rand.Uint64()
		_ = copy(result, nodesQueue)
		for len(nodesQueue) != 0 {
			node := nodesQueue[0]

			if !visited[node.NodeDetails.Id] {
				result = append(result, node)

				client, conn, errClient := createClient(node.NodeDetails.Ip)
				if errClient != nil {
					return nil, errClient
				}

				lookup, errLookup := clientPerformLookup(request.TargetNodeId, magicCookie, client)
				errConn := conn.Close()
				if errConn != nil {
					return nil, errConn
				}

				if errLookup != nil {
					return nil, errLookup
				}

				log.Printf("Node %s returned the following nodes: %v", node.NodeDetails.Id, lookup)
				nodesQueue = append(nodesQueue, lookup...)
				visited[node.NodeDetails.Id] = true
			}

			nodesQueue = nodesQueue[1:]
		}
	}

	if request.Requester != nil {
		updateRoutingTable(bucket, request.Requester)
	}

	log.Println("Lookup finished!")

	if len(result) != 0 {
		sort.Slice(result, func(i, j int) bool {
			return result[i].DistanceToTarget < result[j].DistanceToTarget
		})
		return &ks.LookupResponse{Nodes: result[:k]}, nil
	}

	return &ks.LookupResponse{Nodes: nodesQueue[:k]}, nil
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
	log.Printf("server listening at %s", ip+":"+strconv.Itoa(port))

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
