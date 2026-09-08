package grpcapi

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
)

type ProductServiceServer interface {
	GetProduct(context.Context, *structpb.Struct) (*structpb.Struct, error)
}

type UserServiceServer interface {
	GetUser(context.Context, *structpb.Struct) (*structpb.Struct, error)
}

func RegisterProductServiceServer(server grpc.ServiceRegistrar, implementation ProductServiceServer) {
	server.RegisterService(&ProductService_ServiceDesc, implementation)
}

func RegisterUserServiceServer(server grpc.ServiceRegistrar, implementation UserServiceServer) {
	server.RegisterService(&UserService_ServiceDesc, implementation)
}

func ProductServiceGetProductHandler(server any, ctx context.Context, decoder func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	request := new(structpb.Struct)
	if err := decoder(request); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return server.(ProductServiceServer).GetProduct(ctx, request)
	}

	info := &grpc.UnaryServerInfo{
		Server:     server,
		FullMethod: "/cartnova.product.ProductService/GetProduct",
	}
	handler := func(ctx context.Context, request any) (any, error) {
		return server.(ProductServiceServer).GetProduct(ctx, request.(*structpb.Struct))
	}
	return interceptor(ctx, request, info, handler)
}

func UserServiceGetUserHandler(server any, ctx context.Context, decoder func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	request := new(structpb.Struct)
	if err := decoder(request); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return server.(UserServiceServer).GetUser(ctx, request)
	}

	info := &grpc.UnaryServerInfo{
		Server:     server,
		FullMethod: "/cartnova.user.UserService/GetUser",
	}
	handler := func(ctx context.Context, request any) (any, error) {
		return server.(UserServiceServer).GetUser(ctx, request.(*structpb.Struct))
	}
	return interceptor(ctx, request, info, handler)
}

var ProductService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "cartnova.product.ProductService",
	HandlerType: (*ProductServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetProduct",
			Handler:    ProductServiceGetProductHandler,
		},
	},
}

var UserService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "cartnova.user.UserService",
	HandlerType: (*UserServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetUser",
			Handler:    UserServiceGetUserHandler,
		},
	},
}

func NewProductServiceClient(connection grpc.ClientConnInterface) *ProductServiceClient {
	return &ProductServiceClient{connection: connection}
}

type ProductServiceClient struct {
	connection grpc.ClientConnInterface
}

func (c *ProductServiceClient) GetProduct(ctx context.Context, request *structpb.Struct, options ...grpc.CallOption) (*structpb.Struct, error) {
	response := new(structpb.Struct)
	err := c.connection.Invoke(ctx, "/cartnova.product.ProductService/GetProduct", request, response, options...)
	return response, err
}

func NewUserServiceClient(connection grpc.ClientConnInterface) *UserServiceClient {
	return &UserServiceClient{connection: connection}
}

type UserServiceClient struct {
	connection grpc.ClientConnInterface
}

func (c *UserServiceClient) GetUser(ctx context.Context, request *structpb.Struct, options ...grpc.CallOption) (*structpb.Struct, error) {
	response := new(structpb.Struct)
	err := c.connection.Invoke(ctx, "/cartnova.user.UserService/GetUser", request, response, options...)
	return response, err
}
