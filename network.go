package main

import (
	"log"
	"net"
	"os/exec"
	"strings"
)

var (
	port = 7777
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
			if networkIps[i] == ip || networkIps[i] == "" {
				continue
			}
			log.Printf("Trying to dial %s", networkIps[i])
			client, conn, err := createClient(networkIps[i])
			if err != nil {
				continue
			} else {
				pingResult, errPing := clientPerformPING(client)
				if errPing != nil {
					log.Printf("Error while pinging %v. Moving on to another peer...", networkIps[i])
					continue
				}

				errLookup := clientPerformLookup(id, client)
				if errLookup != nil {
					log.Printf("Error while performing a lookup on %v. Moving on to another peer...", networkIps[i])
					continue
				}

				bucket, errDist := calculateDistance(pingResult.Id, id)
				if errDist != nil {
					return
				}
				updateRoutingTable(bucket, pingResult)
				hasBootstrapped = true
				closeErr := conn.Close()
				if closeErr != nil {
					return
				}
				break
			}
		}
	}

	if !hasBootstrapped {
		log.Println("No peers in network. Will become bootstrap node.")
	}
}
