import os

import grpc

import kademlia_peer_pb2
import kademlia_peer_pb2_grpc



class KademliaClient:
    async def PING(self, container_info: tuple):
        async with grpc.aio.insecure_channel(f"{"127.0.0.1" if os.getenv("LOCAL") == "TRUE" else container_info[0]}:{container_info[2] if os.getenv("LOCAL") == "TRUE" else container_info[1]}") as channel:
            stub = kademlia_peer_pb2_grpc.KademliaServiceStub(channel)
            return await stub.PING(kademlia_peer_pb2.PingCheck())

    async def STORE(self, container_info: tuple, key: str, value: int):
        async with grpc.aio.insecure_channel(f"{"127.0.0.1" if os.getenv("LOCAL") == "TRUE" else container_info[0]}:{container_info[2] if os.getenv("LOCAL") == "TRUE" else container_info[1]}") as channel:
            stub = kademlia_peer_pb2_grpc.KademliaServiceStub(channel)
            return await stub.STORE(kademlia_peer_pb2.StoreRequest(key=key, value=value))

    async def FIND_NODE(self, container_info: tuple, target: str):
        async with grpc.aio.insecure_channel(f"{"127.0.0.1" if os.getenv("LOCAL") == "TRUE" else container_info[0]}:{container_info[2] if os.getenv("LOCAL") == "TRUE" else container_info[1]}") as channel:
            stub = kademlia_peer_pb2_grpc.KademliaServiceStub(channel)
            return await stub.FIND_NODE(kademlia_peer_pb2.LookupRequest(target=target))

    async def FIND_VALUE(self, container_info: tuple, value_id: str):
        async with grpc.aio.insecure_channel(f"{"127.0.0.1" if os.getenv("LOCAL") == "TRUE" else container_info[0]}:{container_info[2] if os.getenv("LOCAL") == "TRUE" else container_info[1]}") as channel:
            stub = kademlia_peer_pb2_grpc.KademliaServiceStub(channel)
            return await stub.FIND_VALUE(kademlia_peer_pb2.LookupRequest(target=value_id))

    async def ROUTING_TABLE_DUMP(self, container_info: tuple):
        async with grpc.aio.insecure_channel(f"{"127.0.0.1" if os.getenv("LOCAL") == "TRUE" else container_info[0]}:{container_info[2] if os.getenv("LOCAL") == "TRUE" else container_info[1]}") as channel:
            stub = kademlia_peer_pb2_grpc.KademliaServiceStub(channel)
            return await stub.ROUTING_TABLE_DUMP(kademlia_peer_pb2.RoutingTableDumpRequest())

    async def DATA_DUMP(self, container_info: tuple):
        async with grpc.aio.insecure_channel(f"{"127.0.0.1" if os.getenv("LOCAL") == "TRUE" else container_info[0]}:{container_info[2] if os.getenv("LOCAL") == "TRUE" else container_info[1]}") as channel:
            stub = kademlia_peer_pb2_grpc.KademliaServiceStub(channel)
            return await stub.DATA_DUMP(kademlia_peer_pb2.DataDumpRequest())