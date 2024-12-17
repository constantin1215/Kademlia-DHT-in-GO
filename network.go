package main

import (
	"flag"
	"net"
)

var (
	port = flag.Int("port", 7777, "The server port")
)

func getIP() string {
	ifaces, _ := net.Interfaces()
	addrs, _ := ifaces[3].Addrs()

	for _, addr := range addrs {
		var ip net.IP
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
