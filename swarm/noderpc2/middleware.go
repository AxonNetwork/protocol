package noderpc

import (
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		log.Infof("[rpc server] req ~> %+v", req)
		log.Infof("[rpc server] info ~> %+v", info)
		return handler(ctx, req)
	}
}

func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		log.Infof("[rpc server] srv ~> %+v", srv)
		log.Infof("[rpc server] info ~> %+v", info)
		err := handler(srv, stream)
		return err
	}
}
