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
	"sync"
	"time"

	"google.golang.org/grpc"
)

var dataLock = sync.Mutex{}

var data = make(map[string]int32)
var dataVersions = make(map[string]int32)
var dataLeaser = make(map[string]string)

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
	filePathData := filepath.Join("/etc/data", os.Getenv("TAG")+"_peer_data.json")
	filePathVersions := filepath.Join("/etc/data", os.Getenv("TAG")+"_peer_data_versions.json")
	dataJsonString, err := json.Marshal(data)
	versionsJsonString, err := json.Marshal(dataVersions)

	if err != nil {
		log.Println("Error marshalling data")
	} else {
		os.WriteFile(filePathData, dataJsonString, 0644)
		os.WriteFile(filePathVersions, versionsJsonString, 0644)
	}
}

func selfHealDataReplicas() {
	// initial delay so the the node's routing table is stable
	time.Sleep(5 * time.Second)
	for {
		for key, value := range data {
			dataLock.Lock()
			leaser, ok := dataLeaser[key]
			if !ok || leaser != config.id {
				dataLock.Unlock()
				continue
			}

			version := dataVersions[key]
			dataLock.Unlock()
			log.Printf("Replicating %v = %v v.%v", key, value, version)
			storeInCluster(&ks.StoreRequest{
				Key:     key,
				Value:   value,
				Version: &version,
				Leaser:  &config.id,
			})
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
		leaseOk := checkLease(request)
		if !leaseOk {
			trackRequester(request.Requester)
			return &ks.StoreResponse{}, nil
		}

		dataLock.Lock()
		_, ok := data[request.Key]
		if !ok {
			data[request.Key] = request.Value
		}

		if request.Version == nil {
			dataVersions[request.Key] = 1
		} else {
			dataVersions[request.Key] = *request.Version
		}

		dataLock.Unlock()

		trackRequester(request.Requester)
		//saveData()

		return &ks.StoreResponse{}, nil
	}

	gatheredNodes, errClosestNodes := findClosestNodes(request.Requester, request.Key, config.routingTable)
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
	replicas := config.k
	if request.Leaser != nil {
		replicas = config.k - 1
	}
	dataNodes := make([]*ks.NodeInfoLookup, 0, config.k)
	for _, node := range resultedNodes {
		client, conn, err := createClient(node.NodeDetails.Ip)
		if err != nil {
			return nil, err
		}

		_, errStore := Store(request.Key, request.Value, magicCookie, client, request.Version, request.Leaser)
		if errStore != nil {
			continue
		}

		errConn := conn.Close()
		if errConn != nil {
			continue
		}

		successes++
		dataNodes = append(dataNodes, node)
		if successes == replicas {
			break
		}
	}

	if len(dataNodes) == 0 {
		return nil, errors.New("Could not store in any node")
	}

	return &ks.StoreResponse{DataNodes: dataNodes}, nil
}

func checkLease(request *ks.StoreRequest) bool {
	if request.Leaser == nil { // if I receive a store without leaser
		dataLock.Lock()
		dataLeaser[request.Key] = config.id // I become the leaser optimistically
		dataLock.Unlock()
		return true
	} else { // tie break  between leasers
		dataLock.Lock()
		myDist, err := calculateDistance(config.id, request.Key)
		reqDist, err := calculateDistance(request.Requester.Id, request.Key)
		if err != nil {
			return false
		}

		if myDist == reqDist {
			if config.id < request.Requester.Id {
				dataLeaser[request.Key] = config.id
			} else {
				dataLeaser[request.Key] = *request.Leaser
			}
		}

		if myDist < reqDist {
			dataLeaser[request.Key] = config.id
		}

		if reqDist < myDist {
			dataLeaser[request.Key] = *request.Leaser
		}

		dataLock.Unlock()
		return true
	}
}

func (s *server) FIND_NODE(_ context.Context, request *ks.LookupRequest) (*ks.LookupResponse, error) {
	log.Printf("FIND_NODE: Request received: %v", request)

	gatheredNodes, errClosestNodes := findClosestNodes(request.Requester, request.Target, config.routingTable)
	if errClosestNodes != nil {
		log.Println("ERROR: Could not find closest nodes.")
		return &ks.LookupResponse{}, nil
	}

	result := make([]*ks.NodeInfoLookup, 0)

	if request.MagicCookie == nil {
		magicCookie := rand.Uint64()
		result = findNodePool(request.Target, magicCookie, gatheredNodes)
	}

	trackRequester(request.Requester)

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

	gatheredNodes, errClosestNodes := findClosestNodes(request.Requester, request.Target, config.routingTable)
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

	trackRequester(request.Requester)

	if foundValue != -1 && foundVersion != 0 {
		return &ks.ValueResponse{Value: &foundValue, Version: &foundVersion}, nil
	}

	if len(result) != 0 {
		return &ks.ValueResponse{Nodes: result[:config.k]}, nil
	}

	return &ks.ValueResponse{Nodes: gatheredNodes}, nil
}

func trackRequester(requester *ks.NodeInfo) {
	if requester != nil {
		requesterBucket, errDist := calculateDistance(requester.Id, config.id)
		if errDist != nil {
			log.Printf("Could not calculate distance to requester.")
		} else {
			updateRoutingTable(requesterBucket, requester, config.routingTable)
		}
	}
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
		Leasers:  dataLeaser,
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
	//go selfHealDataVersion()
	log.Printf("server listening at %s", config.ip+":"+strconv.Itoa(int(config.port)))

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
