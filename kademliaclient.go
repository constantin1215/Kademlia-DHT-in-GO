package main

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
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
		return nil, err
	}

	log.Println("Received node info: ", pingResult)

	return pingResult, nil
}

func clientPerformLookup(id string, client ks.KademliaServiceClient) error {
	log.Println("Performing LOOKUP")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.FIND_NODE(ctx, &ks.LookupRequest{TargetNodeId: id, Requester: &ks.NodeInfo{Ip: ip, Id: id, Port: int32(port)}})
	if err != nil {
		return err
	}

	received := 0
	for {
		node, recvErr := stream.Recv()
		if recvErr == io.EOF {
			log.Printf("Received EOF")
			break
		}
		if node == nil {
			break
		}
		bucket, distErr := calculateDistance(node.Id, id)
		if distErr != nil {
			log.Printf("Error calculating distance: %v", distErr)
			break
		}
		updateRoutingTable(bucket, node)
		received++
	}
	log.Printf("Received %d nodes data", received)

	err = stream.CloseSend()
	if err != nil {
		return err
	}

	return nil
}
