This project focused on implementing the Kademlia Distributed Hash Table (DHT) to solve a core challenge in distributed systems: efficiently locating and retrieving resources in a decentralized network. Kademlia was chosen for its balance of simplicity and efficiency.

The implementation followed the original design from Kademlia: A Peer-to-Peer Information System Based on the XOR Metric by Petar Maymounkov and David Mazieres, with some updates for modern use. The keyspace was expanded to 256 bits using SHA-256 for better security, and RPCs were implemented in Go with gRPC. To test the system, Docker containers simulated a distributed network, allowing for a full evaluation of RPC behavior and the lookup process.
