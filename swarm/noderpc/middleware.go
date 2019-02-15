package noderpc

import (
	"fmt"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/Conscience/protocol/log"
)

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		log.Debugf(trimStr(fmt.Sprintf("[rpc server] %v %+v", info.FullMethod, req), 300))

		x, err := handler(ctx, req)
		if err != nil {
			log.Errorf(trimStr(fmt.Sprintf("[rpc server] %v %+v %+v", info.FullMethod, req, err), 300))
		}

		log.Debugf(trimStr(fmt.Sprintf("[rpc server] %v %+v, %+v", info.FullMethod, req, x), 300))
		return x, err
	}
}

func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		log.Debugf("[rpc server] %v", info.FullMethod)
		err := handler(srv, stream)
		if err != nil {
			log.Errorf("[rpc server] %+v", err)
		}
		return err
	}
}

func trimStr(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
