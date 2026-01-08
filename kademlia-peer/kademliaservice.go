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
	"time"

	"google.golang.org/grpc"
)

var data = make(map[string]int32)
var dataVersions = make(map[string]int32)

func loadData() {
	dataFile, err := os.ReadFile("/etc/data/" + os.Getenv("TAG") + "_peer_data.json")
	if err != nil {
		log.Println("Error reading /etc/data/" + os.Getenv("TAG") + "_peer_data.json")
		return
	}

	versionsFile, err := os.ReadFile("/etc/data/" + os.Getenv("TAG") + "_peer_data_versions.json")
	if err != nil {
		log.Println("Error reading /etc/data/" + os.Getenv("TAG") + "_peer_data_versions.json")
		return
	}

	err = json.Unmarshal(dataFile, &data)
	if err != nil {
		log.Println("Error parsing /etc/data/" + os.Getenv("TAG") + "_peer_data.json")
		return
	}

	err = json.Unmarshal(versionsFile, &dataVersions)
	if err != nil {
		log.Println("Error parsing /etc/data/" + os.Getenv("TAG") + "_peer_data_versions.json")
		data = make(map[string]int32)
		return
	}

	log.Println("Data loaded from /etc/data/" + os.Getenv("TAG") + "_peer_data_versions.json")
	log.Println("Data loaded from /etc/data/" + os.Getenv("TAG") + "_peer_data.json")
}

func selfHealDataReplicas() {
	for {
		for key, value := range data {
			version := dataVersions[key]
			magicCookie := rand.Uint64()
			log.Printf("Replicating %v = %v v.%v", key, value, version)
			storeInCluster(&ks.StoreRequest{
				Key:         key,
				Value:       value,
				MagicCookie: &magicCookie,
				Version:     &version,
			})
			time.Sleep(5 * time.Second)
		}
	}
}

type server struct {
	ks.UnimplementedKademliaServiceServer
}

func (s *server) PING(_ context.Context, _ *ks.PingCheck) (*ks.NodeInfo, error) {
	return &ks.NodeInfo{Ip: config.ip, Port: config.port, Id: config.id}, nil
}

func (s *server) STORE(_ context.Context, request *ks.StoreRequest) (*ks.StoreResponse, error) {
	log.Printf("STORE: Request received: %v", request)

	return storeInCluster(request)
}

func storeInCluster(request *ks.StoreRequest) (*ks.StoreResponse, error) {
	if request.MagicCookie != nil {
		_, ok := data[request.Key]
		if ok {
			// if it has a version attached and it's bigger take that one
			if request.Version != nil && *request.Version > dataVersions[request.Key] {
				data[request.Key] = request.Value
				dataVersions[request.Key] = dataVersions[request.Key]
			} else if request.Version == nil { // otherwise optimistically update it
				data[request.Key] = request.Value
				dataVersions[request.Key] += 1
			}
		} else {
			data[request.Key] = request.Value
			dataVersions[request.Key] = 1
		}

		if request.Requester != nil {
			requesterBucket, errDist := calculateDistance(request.Key, config.id)
			if errDist != nil {
				log.Printf("Could not calculate distance to requester.")
			} else {
				updateRoutingTable(requesterBucket, request.Requester, config.routingTable)
			}
		}

		filePath1 := filepath.Join("/etc/data", os.Getenv("TAG")+"_peer_data.json")
		filePath2 := filepath.Join("/etc/data", os.Getenv("TAG")+"_peer_data_versions.json")
		dataJsonString, err := json.Marshal(data)
		versionsJsonString, err := json.Marshal(dataVersions)
		if err != nil {
			log.Println("Error marshalling data")
		} else {
			os.WriteFile(filePath1, dataJsonString, 0644)
			os.WriteFile(filePath2, versionsJsonString, 0644)
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

		_, errStore := Store(request.Key, request.Value, magicCookie, client, request.Version)
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
		Pairs:    data,
		Versions: dataVersions,
	}, nil
}

func (s *server) HEAL_REPLICAS(_ context.Context, request *ks.HealReplicasRequest) (*ks.HealReplicasResponse, error) {
	if request.MagicCookie != nil && request.Value != nil {
		_, ok := data[request.Key]
		if ok {
			log.Println("Node already has this info!")
			return nil, errors.New("node already has this info")
		}
		data[request.Key] = *request.Value

		if request.Requester != nil {
			requesterBucket, errDist := calculateDistance(request.Key, config.id)
			if errDist != nil {
				log.Printf("Could not calculate distance to requester.")
			} else {
				updateRoutingTable(requesterBucket, request.Requester, config.routingTable)
			}
		}

		filePath1 := filepath.Join("/etc/data", os.Getenv("TAG")+"_peer_data.json")
		filePath2 := filepath.Join("/etc/data", os.Getenv("TAG")+"_peer_data_versions.json")
		dataJsonString, err := json.Marshal(data)
		versionsJsonString, err := json.Marshal(dataVersions)
		if err != nil {
			log.Println("Error marshalling data")
		} else {
			os.WriteFile(filePath1, dataJsonString, 0644)
			os.WriteFile(filePath2, versionsJsonString, 0644)
		}

		return &ks.HealReplicasResponse{}, nil
	}

	value, ok := data[request.Key]
	if !ok {
		log.Printf("ERROR: This node does not have the data to heal.")
		return &ks.HealReplicasResponse{}, nil
	}

	diff := config.k - int(*request.CurrentReplicas)
	log.Printf("HEAL_REPLICAS: Missing %v", diff)

	gatheredNodes, errClosestNodes := findClosestNodes(request.Requester, request.Key)
	if errClosestNodes != nil {
		log.Println("ERROR: Could not find closest nodes.")
		return nil, errors.New("could not find closest nodes")
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

		_, errStore := HealReplicas(request.Key, value, magicCookie, client)
		if errStore != nil {
			continue
		}

		errConn := conn.Close()
		if errConn != nil {
			continue
		}

		successes++
		dataNodes = append(dataNodes, node)
		if successes == diff {
			break
		}
	}

	return &ks.HealReplicasResponse{}, nil
}

func startService() {
	lis, err := net.Listen("tcp4", fmt.Sprintf(":%d", config.port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	ks.RegisterKademliaServiceServer(s, &server{})
	go selfHealDataReplicas()
	log.Printf("server listening at %s", config.ip+":"+strconv.Itoa(int(config.port)))

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
