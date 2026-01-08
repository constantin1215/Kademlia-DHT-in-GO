Run:
```bash
uvicorn main:app --host 0.0.0.0 --port 9000 --reload
```

Compile proto:
```bash
python -m grpc_tools.protoc -I ../kademlia-peer --python_out=. --grpc_python_out=. ../kademlia-peer/kademlia_peer.proto 
```