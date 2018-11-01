package noderpc

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/Conscience/protocol/log"
)

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		x, err := handler(ctx, req)
		if err != nil {
			log.Errorf("[rpc server] %v %+v %v", info.FullMethod, req, err)
		}
		log.Debugf("[rpc server] %v %+v, %+v", info.FullMethod, req, x)
		return x, err
	}
}

func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		log.Debugf("[rpc server] %v", info.FullMethod)
		err := handler(srv, stream)
		if err != nil {
			log.Errorf("[rpc server] %v", err)
		}
		return err
	}
}
