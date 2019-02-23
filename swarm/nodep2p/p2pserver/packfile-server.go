package p2pserver

import (
	"io"

	netp2p "github.com/libp2p/go-libp2p-net"
	"github.com/pkg/errors"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/swarm/nodep2p"
	. "github.com/Conscience/protocol/swarm/wire"
	"github.com/Conscience/protocol/util"
)

func (s *Server) HandlePackfileStreamRequest(stream netp2p.Stream) {
	defer stream.Close()

	req := GetPackfileRequest{}
	err := ReadStructPacket(stream, &req)
	if err != nil {
		log.Errorf("[packfile server] %+v", errors.WithStack(err))
		return
	}
	log.Debugf("[packfile server] incoming handshake")

	// Ensure the client has access
	{
		isAuth, err := s.isAuthorised(req.RepoID, req.Signature)
		if err != nil {
			log.Errorf("[packfile server] %+v", errors.WithStack(err))
			return
		}

		if isAuth == false {
			err := WriteStructPacket(stream, &GetPackfileResponse{ErrUnauthorized: true})
			if err != nil {
				log.Errorf("[packfile server] %+v", errors.WithStack(err))
				return
			}
			return
		}
	}

	// Write the packfile to the stream
	{
		log.Debugf("[packfile server] writing %v objects to packfile stream", len(req.ObjectIDs)/20)

		r := s.node.Repo(req.RepoID)
		if r == nil {
			log.Warnf("[packfile server] cannot find repo %v", req.RepoID)
			err := WriteStructPacket(stream, &GetPackfileResponse{ObjectIDs: []byte{}})
			if err != nil {
				log.Errorf("[packfile server] %+v", errors.WithStack(err))
				return
			}
			return
		}

		availableObjectIDs := [][]byte{}
		for _, id := range UnflattenObjectIDs(req.ObjectIDs) {
			if r.HasObject(id) {
				availableObjectIDs = append(availableObjectIDs, id)
			}
		}

		err = WriteStructPacket(stream, &GetPackfileResponse{ObjectIDs: FlattenObjectIDs(availableObjectIDs)})
		if err != nil {
			log.Errorf("[packfile server] %+v", errors.WithStack(err))
			return
		}

		if len(availableObjectIDs) == 0 {
			return
		}

		rPipe, wPipe := io.Pipe()

		// Generate the packfile
		// @@TODO: to maximize packing efficiency, add commits first, then trees, then blobs.  Among
		// each object type, add most recent objects first.
		go func() {
			var err error
			defer func() { wPipe.CloseWithError(err) }()

			packbuilder, err := r.NewPackbuilder()
			if err != nil {
				log.Errorln("[packfile server] error instantiating packbuilder:", err)
				return
			}

			for i := range availableObjectIDs {
				oid := util.OidFromBytes(availableObjectIDs[i])

				err = packbuilder.Insert(oid, "")
				if err != nil {
					log.Errorln("[packfile server] error adding object to packbuilder:", err)
					return
				}
			}

			err = packbuilder.Write(stream)
			if err != nil {
				log.Errorln("[packfile server] error writing packfile to stream:", err)
				return
			}
		}()

		end := false
		for {
			data := make([]byte, nodep2p.OBJ_CHUNK_SIZE)
			n, err := io.ReadFull(rPipe, data)
			if err == io.EOF {
				break
			} else if err == io.ErrUnexpectedEOF {
				data = data[:n]
			} else if err != nil {
				log.Errorln("[packfile server] error reading packfile")
			}

			if end == true {
				return
			} else {
				err = WriteStructPacket(stream, &GetPackfileResponsePacket{Length: n})
			}

			if err != nil {
				log.Errorf("[packfile server] %+v", errors.WithStack(err))
				return
			}

			n, err = stream.Write(data)
			if err != nil {
				log.Errorf("[packfile server] %+v", errors.WithStack(err))
				return
			}
		}

		err = WriteStructPacket(stream, &GetPackfileResponsePacket{End: true})
		if err != nil {
			log.Errorf("[packfile server] %+v", errors.WithStack(err))
			return
		}
	}
}
