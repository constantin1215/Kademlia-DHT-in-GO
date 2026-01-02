package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"net"
	"os"
	"path/filepath"
	ks "peer/kademlia/service"
	"sort"
	"strconv"

	"google.golang.org/grpc"
)

var data = make(map[string]int32)

func loadData() {
	file, err := os.ReadFile("/etc/data/" + os.Getenv("TAG") + "_peer_data.json")
	if err != nil {
		log.Println("Error reading /etc/data/" + os.Getenv("TAG") + "_peer_data.json")
		return
	}

	err = json.Unmarshal(file, &data)
	if err != nil {
		log.Println("Error parsing /etc/data/" + os.Getenv("TAG") + "_peer_data.json")
	}

	hostname, _ := os.Hostname()
	log.Println(hostname)
	log.Println("Data loaded from /etc/data/" + os.Getenv("TAG") + "_peer_data.json")
}

type server struct {
	ks.UnimplementedKademliaServiceServer
}

func (s *server) PING(_ context.Context, _ *ks.PingCheck) (*ks.NodeInfo, error) {
	return &ks.NodeInfo{Ip: config.ip, Port: config.port, Id: config.id}, nil
}

func (s *server) STORE(_ context.Context, request *ks.StoreRequest) (*ks.StoreResponse, error) {
	log.Printf("STORE: Request received: %v", request)

	if request.MagicCookie != nil {
		data[request.Key] = request.Value

		if request.Requester != nil {
			requesterBucket, errDist := calculateDistance(request.Key, config.id)
			if errDist != nil {
				log.Printf("Could not calculate distance to requester.")
			} else {
				updateRoutingTable(requesterBucket, request.Requester, config.routingTable)
			}
		}

		filePath := filepath.Join("/etc/data", os.Getenv("TAG")+"_peer_data.json")
		dataJsonString, err := json.Marshal(data)
		if err != nil {
			log.Println("Error marshalling data")
		} else {
			os.WriteFile(filePath, dataJsonString, 0644)
		}

		return &ks.StoreResponse{}, nil
	}

	gatheredNodes, errClosestNodes := findClosestNodes(request.Requester, request.Key)
	if errClosestNodes != nil {
		log.Println("ERROR: Could not find closest nodes.")
		return &ks.StoreResponse{}, nil
	}

	result := make(map[string]*ks.NodeInfoLookup)
	magicCookie := rand.Uint64()

	for _, node := range gatheredNodes {
		result[node.NodeDetails.Id] = node
	}
	findNodePool(request.Key, magicCookie, result, gatheredNodes)

	resultedNodes := make([]*ks.NodeInfoLookup, 0, len(result))
	for _, node := range result {
		resultedNodes = append(resultedNodes, node)
	}
	sort.Slice(resultedNodes, func(i, j int) bool {
		return resultedNodes[i].DistanceToTarget < resultedNodes[j].DistanceToTarget
	})

	successes := 0
	dataNodes := make([]*ks.NodeInfoLookup, 0, config.k)
	for _, node := range resultedNodes {
		client, conn, err := createClient(node.NodeDetails.Ip)
		if err != nil {
			return nil, err
		}

		_, errStore := Store(request.Key, request.Value, magicCookie, client)
		if errStore != nil {
			continue
		}

		errConn := conn.Close()
		if errConn != nil {
			continue
		}

		successes++
		dataNodes = append(dataNodes, node)
		if successes == config.k {
			break
		}
	}

	if len(dataNodes) == 0 {
		return nil, errors.New("Could not store in any node")
	}

	return &ks.StoreResponse{DataNodes: dataNodes}, nil
}

func (s *server) FIND_NODE(_ context.Context, request *ks.LookupRequest) (*ks.LookupResponse, error) {
	log.Printf("FIND_NODE: Request received: %v", request)

	gatheredNodes, errClosestNodes := findClosestNodes(request.Requester, request.Target)
	if errClosestNodes != nil {
		log.Println("ERROR: Could not find closest nodes.")
		return &ks.LookupResponse{}, nil
	}

	result := make(map[string]*ks.NodeInfoLookup)

	if request.MagicCookie == nil {
		magicCookie := rand.Uint64()
		for _, node := range gatheredNodes {
			result[node.NodeDetails.Id] = node
		}
		findNodePool(request.Target, magicCookie, result, gatheredNodes)
	}

	if request.Requester != nil {
		requesterBucket, errDist := calculateDistance(request.Target, config.id)
		if errDist != nil {
			log.Printf("Could not calculate distance to requester.")
		} else {
			updateRoutingTable(requesterBucket, request.Requester, config.routingTable)
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
		return &ks.LookupResponse{Nodes: resultedNodes[:config.k]}, nil
	}

	log.Println("Returned gathered nodes")
	return &ks.LookupResponse{Nodes: gatheredNodes}, nil
}

func (s *server) FIND_VALUE(_ context.Context, request *ks.LookupRequest) (*ks.ValueResponse, error) {
	log.Printf("FIND_VALUE: Request received: %v", request)
	value, ok := data[request.Target]
	if ok {
		return &ks.ValueResponse{Value: &value}, nil
	}

	gatheredNodes, errClosestNodes := findClosestNodes(request.Requester, request.Target)
	if errClosestNodes != nil {
		log.Println("ERROR: Could not find closest nodes.")
		return &ks.ValueResponse{}, nil
	}

	result := make(map[string]*ks.NodeInfoLookup)

	var foundValue *int32
	if request.MagicCookie == nil {
		magicCookie := rand.Uint64()
		for _, node := range gatheredNodes {
			result[node.NodeDetails.Id] = node
		}
		findValuePool(request, magicCookie, result, &foundValue, gatheredNodes)
	}

	if request.Requester != nil {
		requesterBucket, errDist := calculateDistance(request.Target, config.id)
		if errDist != nil {
			log.Printf("Could not calculate distance to requester.")
		} else {
			updateRoutingTable(requesterBucket, request.Requester, config.routingTable)
		}
	}

	if foundValue != nil {
		return &ks.ValueResponse{Value: foundValue}, nil
	}

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
		return &ks.ValueResponse{Nodes: resultedNodes[:config.k]}, nil
	}

	return &ks.ValueResponse{Nodes: gatheredNodes}, nil
}

func (s *server) ROUTING_TABLE_DUMP(_ context.Context, _ *ks.RoutingTableDumpRequest) (*ks.RoutingTableDumpResponse, error) {
	convertedRoutingTable := make(map[uint32]*ks.NodeInfoList)

	for key, value := range config.routingTable {
		nodeList := &ks.NodeInfoList{Nodes: value}
		convertedRoutingTable[uint32(key)] = nodeList
	}

	return &ks.RoutingTableDumpResponse{
		Pairs: convertedRoutingTable,
	}, nil
}

func (s *server) DATA_DUMP(_ context.Context, _ *ks.DataDumpRequest) (*ks.DataDumpResponse, error) {
	return &ks.DataDumpResponse{
		Pairs: data,
	}, nil
}

func (s *server) PUT(_ context.Context, request *ks.PutRequest) (*ks.NodeInfo, error) {
	data[request.Key] = request.Value
	return &ks.NodeInfo{Ip: config.ip, Port: config.port, Id: config.id}, nil
}

func startService() {
	lis, err := net.Listen("tcp4", fmt.Sprintf(":%d", config.port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	ks.RegisterKademliaServiceServer(s, &server{})
	log.Printf("server listening at %s", config.ip+":"+strconv.Itoa(int(config.port)))

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
