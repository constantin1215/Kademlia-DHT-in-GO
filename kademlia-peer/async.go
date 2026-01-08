package main

import (
	"fmt"
	"log"
	ks "peer/kademlia/service"
	"sync"
	"sync/atomic"
	"time"
)

const alpha = 3

func WaitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	ch := make(chan struct{})
	go func() {
		wg.Wait()
		close(ch)
	}()
	select {
	case <-ch:
		return true
	case <-time.After(timeout):
		return false
	}
}

func processFindNode(targetId string, magicCookie uint64, result map[string]*ks.NodeInfoLookup, mutexResult *sync.Mutex,
	waitGroup *sync.WaitGroup, nodeChan chan *ks.NodeInfoLookup) {
	defer waitGroup.Done()

	for node := range nodeChan {
		log.Printf("Processing %v", node)

		mutexResult.Lock()
		result[node.NodeDetails.Id] = node
		mutexResult.Unlock()

		client, conn, errClient := createClient(node.NodeDetails.Ip)
		if errClient != nil {
			continue
		}

		lookup, errLookup := LookupNode(targetId, magicCookie, client)
		if errLookup != nil {
			continue
		}

		errConn := conn.Close()
		if errConn != nil {
			continue
		}

		log.Printf("Node %s returned the following nodes: %v", node.NodeDetails.Ip, lookup)
		for _, infoLookup := range lookup {
			mutexResult.Lock()
			_, ok := result[infoLookup.NodeDetails.Id]
			mutexResult.Unlock()
			if !ok {
				go func() {
					nodeChan <- infoLookup
				}()
			}
		}
	}
}

func findNodePool(target string, magicCookie uint64, result map[string]*ks.NodeInfoLookup, gatheredNodes []*ks.NodeInfoLookup) {
	nodeChan := make(chan *ks.NodeInfoLookup, 10)
	var wg sync.WaitGroup
	var mutexResult sync.Mutex

	for i := 0; i < alpha; i++ {
		wg.Add(1)
		go processFindNode(target, magicCookie, result, &mutexResult, &wg, nodeChan)
	}

	for _, node := range gatheredNodes {
		nodeChan <- node
	}

	if WaitTimeout(&wg, 1000*time.Millisecond) {
		fmt.Println("All workers finished within the timeout.")
	} else {
		fmt.Println("Timeout reached before all workers could finish.")
	}
}

func processFindValue(targetKey string, magicCookie uint64, resultedNodes map[string]*ks.NodeInfoLookup, value **int32, mutexResult *sync.Mutex,
	waitGroup *sync.WaitGroup, cancel *atomic.Bool, nodeChan chan *ks.NodeInfoLookup) {
	defer waitGroup.Done()

	for node := range nodeChan {
		log.Printf("Processing %v", node)

		mutexResult.Lock()
		resultedNodes[node.NodeDetails.Id] = node
		mutexResult.Unlock()

		client, conn, errClient := createClient(node.NodeDetails.Ip)
		if errClient != nil {
			continue
		}

		lookup, errLookup := LookupValue(targetKey, magicCookie, client)
		errConn := conn.Close()
		if errConn != nil {
			continue
		}
		if errLookup != nil {
			continue
		}

		if lookup.Value != nil {
			*value = lookup.Value
			cancel.Store(true)
			break
		}

		log.Printf("Node %s returned the following nodes: %v", node.NodeDetails.Ip, lookup)
		for _, infoLookup := range lookup.Nodes {
			mutexResult.Lock()
			_, ok := resultedNodes[infoLookup.NodeDetails.Id]
			mutexResult.Unlock()
			if !ok {
				go func() {
					nodeChan <- infoLookup
					log.Printf("(Added to chan) Node %s sent the following info: %v", node.NodeDetails.Ip, infoLookup)
				}()
			}
		}

		if cancel.Load() {
			break
		}
	}
}

func findValuePool(request *ks.LookupRequest, magicCookie uint64, result map[string]*ks.NodeInfoLookup, value **int32, gatheredNodes []*ks.NodeInfoLookup) {
	nodeChan := make(chan *ks.NodeInfoLookup, 10)
	var wg sync.WaitGroup
	var mutexResult sync.Mutex
	cancel := atomic.Bool{}
	cancel.Store(false)

	for i := 0; i < alpha; i++ {
		wg.Add(1)
		go processFindValue(request.Target, magicCookie, result, value, &mutexResult, &wg, &cancel, nodeChan)
	}

	for _, node := range gatheredNodes {
		nodeChan <- node
	}

	if WaitTimeout(&wg, 100*time.Millisecond) {
		fmt.Println("All workers finished within the timeout.")
	} else {
		fmt.Println("Timeout reached before all workers could finish.")
	}
}
