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
	"sync"
	"time"
)

var (
	id           = calculateNodeId()
	routingTable = initRoutingTable()
	data         = make(map[int32]int32)
)

const (
	k     = 4
	alpha = 3
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

func WaitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	ch := make(chan struct{})
	go func() {
		wg.Wait()
		close(ch)
	}()
	select {
	case <-ch:
		return true
	case <-time.After(timeout):
		return false
	}
}

func processNode(targetId string, magicCookie uint64, result map[string]*ks.NodeInfoLookup, mutexResult *sync.Mutex,
	waitGroup *sync.WaitGroup, nodeChan chan *ks.NodeInfoLookup) {
	defer waitGroup.Done()

	for node := range nodeChan {
		log.Printf("Processing %v", node)

		mutexResult.Lock()
		result[node.NodeDetails.Id] = node
		mutexResult.Unlock()

		client, conn, errClient := createClient(node.NodeDetails.Ip)
		if errClient != nil {
			continue
		}

		lookup, errLookup := clientPerformLookup(targetId, magicCookie, client)
		errConn := conn.Close()
		if errConn != nil {
			continue
		}

		if errLookup != nil {
			continue
		}

		log.Printf("Node %s returned the following nodes: %v", node.NodeDetails.Ip, lookup)
		for _, infoLookup := range lookup {
			mutexResult.Lock()
			_, ok := result[infoLookup.NodeDetails.Id]
			mutexResult.Unlock()
			if !ok {
				go func() {
					nodeChan <- infoLookup
					log.Printf("(Added to chan) Node %s sent the following info: %v", node.NodeDetails.Ip, infoLookup)
				}()
			}
		}
	}
}

func (s *server) FIND_NODE(_ context.Context, request *ks.LookupRequest) (*ks.LookupResponse, error) {
	log.Printf("FIND_NODE: Request received: %v", request)

	bucket, err := calculateDistance(request.TargetNodeId, id)
	if err != nil {
		fmt.Println("Error calculating distance, invalid character detected in node ID.")
		return nil, err
	}

	var gatheredNodes []*ks.NodeInfoLookup
	result := make(map[string]*ks.NodeInfoLookup)

	if request.Requester != nil {
		gatheredNodes = gatherClosestNodes(bucket, request.Requester.Id, request.TargetNodeId)
	} else {
		gatheredNodes = gatherClosestNodes(bucket, "", request.TargetNodeId)
	}
	log.Printf("Gathered nodes %v", gatheredNodes)

	if request.MagicCookie == nil {
		magicCookie := rand.Uint64()
		for _, node := range gatheredNodes {
			result[node.NodeDetails.Id] = node
		}
		nodeChan := make(chan *ks.NodeInfoLookup, 10)
		var wg sync.WaitGroup
		var mutexResult sync.Mutex

		for i := 0; i < alpha; i++ {
			wg.Add(1)
			go processNode(request.TargetNodeId, magicCookie, result, &mutexResult, &wg, nodeChan)
		}

		for _, node := range gatheredNodes {
			nodeChan <- node
		}

		if WaitTimeout(&wg, 100*time.Millisecond) {
			fmt.Println("All workers finished within the timeout.")
		} else {
			fmt.Println("Timeout reached before all workers could finish.")
		}
	}

	if request.Requester != nil {
		requesterBucket, errDist := calculateDistance(request.TargetNodeId, id)
		if errDist != nil {
			log.Printf("Could not calculate distance to requester.")
		} else {
			updateRoutingTable(requesterBucket, request.Requester)
		}
	}

	log.Println("Lookup finished!")

	if len(result) != 0 {
		//this implementation is shit
		resultedNodes := make([]*ks.NodeInfoLookup, 0, len(result))
		for _, node := range result {
			resultedNodes = append(resultedNodes, node)
		}
		sort.Slice(resultedNodes, func(i, j int) bool {
			return resultedNodes[i].DistanceToTarget < resultedNodes[j].DistanceToTarget
		})
		log.Println("Returned result")
		return &ks.LookupResponse{Nodes: resultedNodes[:k]}, nil
	}

	log.Println("Returned gathered nodes")
	return &ks.LookupResponse{Nodes: gatheredNodes}, nil
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
