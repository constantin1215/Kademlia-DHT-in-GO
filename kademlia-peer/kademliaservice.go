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

func saveData() {
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
}

func selfHealDataReplicas() {
	for {
		for key, value := range data {
			version := dataVersions[key]
			log.Printf("Replicating %v = %v v.%v", key, value, version)
			storeInCluster(&ks.StoreRequest{
				Key:     key,
				Value:   value,
				Version: &version,
			})
			time.Sleep(1 * time.Second)
		}
	}
}

func selfHealDataVersion() {
	for {
		for key, value := range data {
			version := dataVersions[key]
			log.Printf("Checking data %v = %v v.%v in the cluster for newer versions", key, value, version)
			response, err := findInCluster(&ks.LookupRequest{
				Target: key,
			})
			if err != nil {
				continue
			}
			if *response.Version > version {
				dataVersions[key] = version
				data[key] = value
			}
			saveData()
			time.Sleep(1 * time.Second)
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
				dataVersions[request.Key] = *request.Version
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

		saveData()

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
	resultedNodes := findNodePool(request.Key, magicCookie, gatheredNodes)

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

	result := make([]*ks.NodeInfoLookup, 0)

	if request.MagicCookie == nil {
		magicCookie := rand.Uint64()
		result = findNodePool(request.Target, magicCookie, gatheredNodes)
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
		return &ks.LookupResponse{Nodes: result[:config.k]}, nil
	}

	log.Println("Returned gathered nodes")
	return &ks.LookupResponse{Nodes: gatheredNodes}, nil
}

func (s *server) FIND_VALUE(_ context.Context, request *ks.LookupRequest) (*ks.ValueResponse, error) {
	log.Printf("FIND_VALUE: Request received: %v", request)

	return findInCluster(request)
}

func findInCluster(request *ks.LookupRequest) (*ks.ValueResponse, error) {
	value, okData := data[request.Target]
	version, okVersion := dataVersions[request.Target]
	if okData && okVersion {
		return &ks.ValueResponse{Value: &value, Version: &version}, nil
	}

	gatheredNodes, errClosestNodes := findClosestNodes(request.Requester, request.Target)
	if errClosestNodes != nil {
		log.Println("ERROR: Could not find closest nodes.")
		return &ks.ValueResponse{}, nil
	}

	result := make([]*ks.NodeInfoLookup, 0)
	var foundValue int32 = -1
	var foundVersion int32 = 0

	if request.MagicCookie == nil {
		magicCookie := rand.Uint64()
		for _, node := range gatheredNodes {
			result = append(result, node)
		}
		result, foundValue, foundVersion = findValuePool(request, magicCookie, gatheredNodes)
	}

	if request.Requester != nil {
		requesterBucket, errDist := calculateDistance(request.Target, config.id)
		if errDist != nil {
			log.Printf("Could not calculate distance to requester.")
		} else {
			updateRoutingTable(requesterBucket, request.Requester, config.routingTable)
		}
	}

	if foundValue != -1 && foundVersion != 0 {
		return &ks.ValueResponse{Value: &foundValue, Version: &foundVersion}, nil
	}

	if len(result) != 0 {
		return &ks.ValueResponse{Nodes: result[:config.k]}, nil
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

func startService() {
	lis, err := net.Listen("tcp4", fmt.Sprintf(":%d", config.port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	ks.RegisterKademliaServiceServer(s, &server{})
	go selfHealDataReplicas()
	go selfHealDataVersion()
	log.Printf("server listening at %s", config.ip+":"+strconv.Itoa(int(config.port)))

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
