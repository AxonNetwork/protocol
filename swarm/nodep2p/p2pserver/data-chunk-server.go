package p2pserver

// import (
// 	netp2p "github.com/libp2p/go-libp2p-net"

// 	"github.com/Conscience/protocol/log"
// 	. "github.com/Conscience/protocol/swarm/wire"
// )

// func (s *Server) HandleDataChunkHandshake(stream netp2p.Stream) {
// 	req := DataChunkHandshakeRequest{}
// 	err := ReadStructPacket{stream, &req}
// 	if err != nil {
// 		log.Errorf("[p2p server] %v", err)
// 		return
// 	}
// 	log.Debugf("[p2p server] incoming handshake")
// 	// Ensure the client has access
// 	{
// 		isAuth, err := s.isAuthorised(req.RepoID, req.Signature)
// 		if err != nil {
// 			log.Errorf("[p2p server] %v", err)
// 			return
// 		}

// 		if isAuth == false {
// 			log.Warnf("[p2p server] address 0x%0x does not have pull access", addr.Bytes())
// 			err := WriteStructPacket(stream, &DataChunkHandshakeResponse{ErrUnauthorized: true})
// 			if err != nil {
// 				log.Errorf("[p2p server] %v", err)
// 				return
// 			}
// 			return
// 		}
// 	}
// 	err = WriteStructPacket(stream, &DataChunkHandshakeResponse{})
// 	if err != nil {
// 		log.Errorf("[p2p server] %v", err)
// 		return
// 	}
// 	go func() {

// 	}()
// }
