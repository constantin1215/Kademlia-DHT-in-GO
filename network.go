package main

import (
	"log"
	"math/rand/v2"
	"net"
	"os/exec"
	ks "peer/kademlia/service"
	"strings"
)

var (
	port = int32(7777)
	ip   = getIP()
)

func getIP() string {
	ifaces, _ := net.Interfaces()
	addrs, _ := ifaces[len(ifaces)-1].Addrs() // last interface is the exposed one
	var ip net.IP
	for _, addr := range addrs {
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		return ip.String()
	}

	return "0.0.0.0"
}

func scanNetwork() ([]string, error) {
	cmd := "nmap -n " + ip + "/28" + " | grep 'Nmap scan report for' | cut -f 5 -d ' '"
	executor := exec.Command("bash", "-c", cmd)
	output, err := executor.Output()

	if err != nil {
		return []string{}, err
	}

	ips := strings.Split(string(output), "\n")

	return ips, nil
}

func joinNetwork() {
	log.Println("Joining network...")
	networkIps, err := scanNetwork()
	if err != nil {
		log.Fatalf("Error while scanning the network! %v", err)
		return
	}

	log.Println("Network peers:", networkIps)
	hasBootstrapped := false

	if len(networkIps) != 0 {
		for i := range networkIps {
			// networkIps[0] == networkIps[i] is meant to skip the first IP because that one is the gateway
			if networkIps[0] == networkIps[i] || networkIps[i] == ip || networkIps[i] == "" {
				continue
			}
			log.Printf("Trying to dial %s", networkIps[i])
			client, conn, errClient := createClient(networkIps[i])
			if errClient != nil {
				continue
			} else {
				pingResult, errPing := clientPerformPING(client)
				if errPing != nil {
					log.Printf("Error while pinging %v. Moving on to another peer...", networkIps[i])
					continue
				}
				magicCookie := rand.Uint64()
				nodes, errLookup := clientPerformLookupNode(id, magicCookie, client)
				closeErr := conn.Close()
				if closeErr != nil {
					return
				}

				if errLookup != nil {
					log.Printf("Error while performing a lookup on %v. Moving on to another peer...", networkIps[i])
					continue
				}

				log.Printf("Received nodes data %v", nodes)
				neighboursResults := make([]*ks.NodeInfoLookup, 0, len(nodes))
				for _, node := range nodes {
					newNodeClient, newNodeConn, newNodeClientErr := createClient(node.NodeDetails.Ip)
					if newNodeClientErr != nil {
						continue
					}
					newNodeLookupResult, newNodeLookupErr := clientPerformLookupNode(id, magicCookie, newNodeClient)
					if newNodeLookupErr != nil {
						continue
					}
					neighboursResults = append(neighboursResults, newNodeLookupResult...)
					closeNewNodeConnErr := newNodeConn.Close()
					if closeNewNodeConnErr != nil {
						continue
					}
					updateRoutingTable(uint16(node.DistanceToTarget), node.NodeDetails)
				}

				for _, node := range neighboursResults {
					updateRoutingTable(uint16(node.DistanceToTarget), node.NodeDetails)
				}

				bucket, errDist := calculateDistance(pingResult.Id, id)
				if errDist != nil {
					return
				}

				updateRoutingTable(bucket, pingResult)
				hasBootstrapped = true
				//log.Printf("My routing table is %v", routingTable)

				log.Println("Successfully joined the network.")
				break
			}
		}
	}

	if !hasBootstrapped {
		log.Println("No peers in network. Will become bootstrap node.")
	}
}
