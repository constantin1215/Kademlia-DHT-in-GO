package main

import (
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
	addrs, _ := ifaces[3].Addrs()
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
	cmd := "nmap -sL -n " + ip + "/28" + " | grep 'Nmap scan report for' | cut -f 5 -d ' '"
	executor := exec.Command("bash", "-c", cmd)
	output, err := executor.Output()

	if err != nil {
		return []string{}, err
	}

	ips := strings.Split(string(output), "\n")

	return ips, nil
}
