package main

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	ks "peer/kademlia/service"
	"time"
)

func createClient(peerIp string) (ks.KademliaServiceClient, *grpc.ClientConn, error) {
	log.Printf("Connecting to node at %s", peerIp)
	var opts []grpc.DialOption

	//add TLS later
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.NewClient(peerIp+":7777", opts...)
	if err != nil {
		log.Printf("failed to dial: %v", err)
		return nil, nil, err
	}

	client := ks.NewKademliaServiceClient(conn)

	return client, conn, nil
}

func clientPerformPING(client ks.KademliaServiceClient) (*ks.NodeInfo, error) {
	log.Println("Performing PING")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pingResult, err := client.PING(ctx, &ks.PingCheck{})
	if err != nil {
		log.Println("Failed to perform PING")
		return nil, err
	}

	log.Println("Received node info: ", pingResult)

	return pingResult, nil
}

func clientPerformLookupNode(targetNodeId string, magicCookie uint64, client ks.KademliaServiceClient) ([]*ks.NodeInfoLookup, error) {
	log.Println("Performing NODE LOOKUP")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := client.FIND_NODE(ctx, &ks.LookupRequest{Target: targetNodeId, Requester: &ks.NodeInfo{Ip: ip, Id: id, Port: port}, MagicCookie: &magicCookie})
	if err != nil {
		return nil, err
	}

	return response.Nodes, nil
}

func clientPerformLookupValue(targetKey string, magicCookie uint64, client ks.KademliaServiceClient) (*ks.ValueResponse, error) {
	log.Println("Performing VALUE LOOKUP")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := client.FIND_VALUE(ctx, &ks.LookupRequest{Target: targetKey, Requester: &ks.NodeInfo{Ip: ip, Id: id, Port: port}, MagicCookie: &magicCookie})
	if err != nil {
		return nil, err
	}

	return response, nil
}

func clientPerformStore(key string, value int32, magicCookie uint64, client ks.KademliaServiceClient) (*ks.StoreResponse, error) {
	log.Println("Performing STORE")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := client.STORE(ctx, &ks.StoreRequest{Key: key, Value: value, Requester: &ks.NodeInfo{Ip: ip, Id: id, Port: port}, MagicCookie: &magicCookie})
	if err != nil {
		return nil, err
	}

	return response, nil
}
