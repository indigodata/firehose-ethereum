// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.21.12
// source: sf/ethereum/indigo/trxhash.proto

package pbtrxhash

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// TransactionHashStreamClient is the client API for TransactionHashStream service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type TransactionHashStreamClient interface {
	Hashes(ctx context.Context, in *HashRequest, opts ...grpc.CallOption) (TransactionHashStream_HashesClient, error)
}

type transactionHashStreamClient struct {
	cc grpc.ClientConnInterface
}

func NewTransactionHashStreamClient(cc grpc.ClientConnInterface) TransactionHashStreamClient {
	return &transactionHashStreamClient{cc}
}

func (c *transactionHashStreamClient) Hashes(ctx context.Context, in *HashRequest, opts ...grpc.CallOption) (TransactionHashStream_HashesClient, error) {
	stream, err := c.cc.NewStream(ctx, &TransactionHashStream_ServiceDesc.Streams[0], "/sf.ethereum.indigo.TransactionHashStream/Hashes", opts...)
	if err != nil {
		return nil, err
	}
	x := &transactionHashStreamHashesClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type TransactionHashStream_HashesClient interface {
	Recv() (*TransactionHash, error)
	grpc.ClientStream
}

type transactionHashStreamHashesClient struct {
	grpc.ClientStream
}

func (x *transactionHashStreamHashesClient) Recv() (*TransactionHash, error) {
	m := new(TransactionHash)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// TransactionHashStreamServer is the server API for TransactionHashStream service.
// All implementations should embed UnimplementedTransactionHashStreamServer
// for forward compatibility
type TransactionHashStreamServer interface {
	Hashes(*HashRequest, TransactionHashStream_HashesServer) error
}

// UnimplementedTransactionHashStreamServer should be embedded to have forward compatible implementations.
type UnimplementedTransactionHashStreamServer struct {
}

func (UnimplementedTransactionHashStreamServer) Hashes(*HashRequest, TransactionHashStream_HashesServer) error {
	return status.Errorf(codes.Unimplemented, "method Hashes not implemented")
}

// UnsafeTransactionHashStreamServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to TransactionHashStreamServer will
// result in compilation errors.
type UnsafeTransactionHashStreamServer interface {
	mustEmbedUnimplementedTransactionHashStreamServer()
}

func RegisterTransactionHashStreamServer(s grpc.ServiceRegistrar, srv TransactionHashStreamServer) {
	s.RegisterService(&TransactionHashStream_ServiceDesc, srv)
}

func _TransactionHashStream_Hashes_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(HashRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(TransactionHashStreamServer).Hashes(m, &transactionHashStreamHashesServer{stream})
}

type TransactionHashStream_HashesServer interface {
	Send(*TransactionHash) error
	grpc.ServerStream
}

type transactionHashStreamHashesServer struct {
	grpc.ServerStream
}

func (x *transactionHashStreamHashesServer) Send(m *TransactionHash) error {
	return x.ServerStream.SendMsg(m)
}

// TransactionHashStream_ServiceDesc is the grpc.ServiceDesc for TransactionHashStream service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var TransactionHashStream_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "sf.ethereum.indigo.TransactionHashStream",
	HandlerType: (*TransactionHashStreamServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Hashes",
			Handler:       _TransactionHashStream_Hashes_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "sf/ethereum/indigo/trxhash.proto",
}
