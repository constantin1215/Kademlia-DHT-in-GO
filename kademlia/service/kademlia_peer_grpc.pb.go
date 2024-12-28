// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.27.2
// source: kademlia_peer.proto

package service

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	KademliaService_PING_FullMethodName       = "/KademliaService/PING"
	KademliaService_STORE_FullMethodName      = "/KademliaService/STORE"
	KademliaService_FIND_NODE_FullMethodName  = "/KademliaService/FIND_NODE"
	KademliaService_FIND_VALUE_FullMethodName = "/KademliaService/FIND_VALUE"
)

// KademliaServiceClient is the client API for KademliaService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type KademliaServiceClient interface {
	PING(ctx context.Context, in *PingCheck, opts ...grpc.CallOption) (*NodeInfo, error)
	STORE(ctx context.Context, in *StoreRequest, opts ...grpc.CallOption) (*StoreResult, error)
	FIND_NODE(ctx context.Context, in *LookupRequest, opts ...grpc.CallOption) (*LookupResponse, error)
	FIND_VALUE(ctx context.Context, in *Key, opts ...grpc.CallOption) (*Value, error)
}

type kademliaServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewKademliaServiceClient(cc grpc.ClientConnInterface) KademliaServiceClient {
	return &kademliaServiceClient{cc}
}

func (c *kademliaServiceClient) PING(ctx context.Context, in *PingCheck, opts ...grpc.CallOption) (*NodeInfo, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(NodeInfo)
	err := c.cc.Invoke(ctx, KademliaService_PING_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *kademliaServiceClient) STORE(ctx context.Context, in *StoreRequest, opts ...grpc.CallOption) (*StoreResult, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(StoreResult)
	err := c.cc.Invoke(ctx, KademliaService_STORE_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *kademliaServiceClient) FIND_NODE(ctx context.Context, in *LookupRequest, opts ...grpc.CallOption) (*LookupResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(LookupResponse)
	err := c.cc.Invoke(ctx, KademliaService_FIND_NODE_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *kademliaServiceClient) FIND_VALUE(ctx context.Context, in *Key, opts ...grpc.CallOption) (*Value, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Value)
	err := c.cc.Invoke(ctx, KademliaService_FIND_VALUE_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// KademliaServiceServer is the server API for KademliaService service.
// All implementations must embed UnimplementedKademliaServiceServer
// for forward compatibility.
type KademliaServiceServer interface {
	PING(context.Context, *PingCheck) (*NodeInfo, error)
	STORE(context.Context, *StoreRequest) (*StoreResult, error)
	FIND_NODE(context.Context, *LookupRequest) (*LookupResponse, error)
	FIND_VALUE(context.Context, *Key) (*Value, error)
	mustEmbedUnimplementedKademliaServiceServer()
}

// UnimplementedKademliaServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedKademliaServiceServer struct{}

func (UnimplementedKademliaServiceServer) PING(context.Context, *PingCheck) (*NodeInfo, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PING not implemented")
}
func (UnimplementedKademliaServiceServer) STORE(context.Context, *StoreRequest) (*StoreResult, error) {
	return nil, status.Errorf(codes.Unimplemented, "method STORE not implemented")
}
func (UnimplementedKademliaServiceServer) FIND_NODE(context.Context, *LookupRequest) (*LookupResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method FIND_NODE not implemented")
}
func (UnimplementedKademliaServiceServer) FIND_VALUE(context.Context, *Key) (*Value, error) {
	return nil, status.Errorf(codes.Unimplemented, "method FIND_VALUE not implemented")
}
func (UnimplementedKademliaServiceServer) mustEmbedUnimplementedKademliaServiceServer() {}
func (UnimplementedKademliaServiceServer) testEmbeddedByValue()                         {}

// UnsafeKademliaServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to KademliaServiceServer will
// result in compilation errors.
type UnsafeKademliaServiceServer interface {
	mustEmbedUnimplementedKademliaServiceServer()
}

func RegisterKademliaServiceServer(s grpc.ServiceRegistrar, srv KademliaServiceServer) {
	// If the following call pancis, it indicates UnimplementedKademliaServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&KademliaService_ServiceDesc, srv)
}

func _KademliaService_PING_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PingCheck)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(KademliaServiceServer).PING(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: KademliaService_PING_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(KademliaServiceServer).PING(ctx, req.(*PingCheck))
	}
	return interceptor(ctx, in, info, handler)
}

func _KademliaService_STORE_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StoreRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(KademliaServiceServer).STORE(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: KademliaService_STORE_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(KademliaServiceServer).STORE(ctx, req.(*StoreRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _KademliaService_FIND_NODE_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(LookupRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(KademliaServiceServer).FIND_NODE(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: KademliaService_FIND_NODE_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(KademliaServiceServer).FIND_NODE(ctx, req.(*LookupRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _KademliaService_FIND_VALUE_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Key)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(KademliaServiceServer).FIND_VALUE(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: KademliaService_FIND_VALUE_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(KademliaServiceServer).FIND_VALUE(ctx, req.(*Key))
	}
	return interceptor(ctx, in, info, handler)
}

// KademliaService_ServiceDesc is the grpc.ServiceDesc for KademliaService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var KademliaService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "KademliaService",
	HandlerType: (*KademliaServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "PING",
			Handler:    _KademliaService_PING_Handler,
		},
		{
			MethodName: "STORE",
			Handler:    _KademliaService_STORE_Handler,
		},
		{
			MethodName: "FIND_NODE",
			Handler:    _KademliaService_FIND_NODE_Handler,
		},
		{
			MethodName: "FIND_VALUE",
			Handler:    _KademliaService_FIND_VALUE_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "kademlia_peer.proto",
}
