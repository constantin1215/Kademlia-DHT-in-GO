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
var dataRefreshTime = make(map[string]int64)

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
	time.Sleep(5 * time.Second)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		dataLock.Lock()
		keys := make([]string, 0, len(data))
		for key := range data {
			keys = append(keys, key)
		}
		dataLock.Unlock()

		for _, key := range keys {
			dataLock.Lock()
			value, ok := data[key]
			if !ok {
				dataLock.Unlock()
				continue
			}

			leaser, hasLeaser := dataLeaser[key]
			refreshTime, hasRefresh := dataRefreshTime[key]
			dataLock.Unlock()

			now := time.Now().UnixMilli()
			timeout := int64(30_000)
			shouldClaim := false

			log.Printf("Will timeout in %d ms", timeout-(now-refreshTime))
			if !hasLeaser || leaser == "" {
				shouldClaim = true
			} else if hasRefresh && (now-refreshTime) > timeout {
				shouldClaim = true
			} else {
				myDist, err1 := calculateDistance(config.id, key)
				leaserDist, err2 := calculateDistance(leaser, key)
				if err1 == nil && err2 == nil {
					if myDist < leaserDist {
						shouldClaim = true
					} else if myDist == leaserDist && config.id < leaser {
						shouldClaim = true
					}
				}
			}

			if shouldClaim {
				dataLock.Lock()
				dataLeaser[key] = config.id
				dataLock.Unlock()
				leaser = config.id
			}

			if leaser != config.id {
				continue
			}

			dataLock.Lock()
			version := dataVersions[key]
			dataLock.Unlock()

			log.Printf("Replicating %v = %v v.%v", key, value, version)
			_, _ = storeInCluster(&ks.StoreRequest{
				Key:     key,
				Value:   value,
				Version: &version,
				Leaser:  &config.id,
			})
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
		dataRefreshTime[request.Key] = time.Now().UnixMilli()
		leaseOk := checkLease(request)
		if !leaseOk {
			trackRequester(request.Requester)
			return &ks.StoreResponse{}, nil
		}

		var currentVersion int32

		dataLock.Lock()
		defer dataLock.Unlock()
		if dataLeaser[request.Key] == config.id {
			if request.Version == nil {
				if v, ok := dataVersions[request.Key]; ok {
					currentVersion = v + 1
				} else {
					currentVersion = 1
				}
			} else {
				currentVersion = *request.Version
			}
			dataVersions[request.Key] = currentVersion
			data[request.Key] = request.Value
		} else {
			if request.Version == nil {
				return &ks.StoreResponse{}, nil
			}

			currentVersion = *request.Version
			existingVersion, hasVersion := dataVersions[request.Key]

			if hasVersion && currentVersion < existingVersion {
				return &ks.StoreResponse{}, nil
			}

			dataVersions[request.Key] = currentVersion
			data[request.Key] = request.Value
		}

		trackRequester(request.Requester)
		saveData()

		return &ks.StoreResponse{}, nil
	}

	gatheredNodes, errClosestNodes := findClosestNodes(request.Requester, request.Key, config.routingTable)
	if errClosestNodes != nil || len(gatheredNodes) == 0 {
		log.Println("ERROR: Could not find closest nodes.")
		return &ks.StoreResponse{}, nil
	}

	magicCookie := rand.Uint64()
	resultedNodes := findNodePool(request.Key, magicCookie, gatheredNodes)

	var selectedLeaser string
	if request.Leaser != nil {
		selectedLeaser = *request.Leaser
	} else {
		closestNode := &ks.NodeInfo{Id: config.id, Ip: config.ip}
		minDist := uint16(256)
		for _, node := range resultedNodes {
			dist, err := calculateDistance(node.NodeDetails.Id, request.Key)
			if err != nil {
				continue
			}
			if dist < minDist || (dist == minDist && node.NodeDetails.Id < closestNode.Id) {
				minDist = dist
				closestNode = node.NodeDetails
			}
		}
		selectedLeaser = closestNode.Id
	}

	successes := 0
	replicas := config.k
	dataNodes := make([]*ks.NodeInfoLookup, 0, config.k)

	for _, node := range resultedNodes {
		if successes >= replicas {
			break
		}

		if node.NodeDetails.Id == config.id {
			successes++
			dataNodes = append(dataNodes, node)
			continue
		}

		client, conn, err := createClient(node.NodeDetails.Ip)
		if err != nil {
			continue
		}

		_, errStore := Store(
			request.Key,
			request.Value,
			magicCookie,
			client,
			request.Version,
			&selectedLeaser,
		)
		_ = conn.Close()

		if errStore == nil {
			successes++
			dataNodes = append(dataNodes, node)
		}
	}

	if successes == 0 {
		return nil, errors.New("store failed on all nodes")
	}

	return &ks.StoreResponse{DataNodes: dataNodes}, nil
}

func checkLease(request *ks.StoreRequest) bool {
	if request.Leaser == nil || request.Requester == nil {
		return false
	}

	key := request.Key
	incomingLeaser := *request.Leaser

	log.Printf("Checking incoming leaser %v", incomingLeaser)

	dataLock.Lock()
	currentLeaser, hasLeaser := dataLeaser[key]
	dataLock.Unlock()

	if hasLeaser {
		log.Printf("Current leaser %v", currentLeaser)
	}

	if !hasLeaser || currentLeaser == "" {
		dataLock.Lock()
		dataLeaser[key] = incomingLeaser
		dataLock.Unlock()
		log.Printf("Set leaser to %v", incomingLeaser)
		return true
	}

	if incomingLeaser == currentLeaser {
		return true
	}

	currentDist, err1 := calculateDistance(currentLeaser, key)
	incomingDist, err2 := calculateDistance(incomingLeaser, key)
	if err1 != nil || err2 != nil {
		return false
	}

	log.Printf("Comparing incoming leaser dist %v with current leaser dist %v", incomingDist, currentDist)

	if incomingDist < currentDist || (incomingDist == currentDist && incomingLeaser < currentLeaser) {

		log.Printf("Yielding key %s: %s -> %s", key, currentLeaser, incomingLeaser)

		dataLock.Lock()
		delete(data, key)
		delete(dataVersions, key)
		delete(dataRefreshTime, key)
		delete(dataLeaser, key)
		dataLock.Unlock()

		return true
	}

	log.Printf("Forcing yield of false leaser %s (real %s) for key %s", incomingLeaser, currentLeaser, key)

	client, conn, err := createClient(request.Requester.Ip)
	if err == nil {
		defer conn.Close()

		magicCookie := rand.Uint64()

		_, _ = Store(
			key,
			0,
			magicCookie,
			client,
			nil,
			&currentLeaser,
		)
		return false
	}

	return false
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

	log.Printf("server listening at %s", config.ip+":"+strconv.Itoa(int(config.port)))

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
