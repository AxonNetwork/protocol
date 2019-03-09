package nodep2p

import (
	"io"

	netp2p "github.com/libp2p/go-libp2p-net"
	"github.com/pkg/errors"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/repo"
	"github.com/Conscience/protocol/swarm/wire"
	"github.com/Conscience/protocol/util"
)

func (s *Server) HandlePackfileStreamRequest(stream netp2p.Stream) {
	defer stream.Close()

	for {
		req := wire.GetPackfileRequest{}
		err := wire.ReadStructPacket(stream, &req)
		if err == io.EOF {
			log.Debugf("[packfile server] peer closed stream")
			return
		} else if err != nil {
			log.Errorf("[packfile server] %+v", errors.WithStack(err))
			return
		}
		log.Debugf("[packfile server] incoming handshake")

		// Ensure the client has access, then report which of the objectIDs we can provide.
		var availableObjectIDs [][]byte
		var r *repo.Repo
		{
			isAuth, err := s.isAuthorised(req.RepoID, req.Signature)
			if err != nil {
				log.Errorf("[packfile server] %+v", errors.WithStack(err))
				return
			}

			if isAuth == false {
				err := wire.WriteStructPacket(stream, &wire.GetPackfileResponseHeader{ErrUnauthorized: true})
				if err != nil {
					log.Errorf("[packfile server] %+v", errors.WithStack(err))
					return
				}
				return
			}

			r = s.node.Repo(req.RepoID)
			if r == nil {
				log.Warnf("[packfile server] cannot find repo %v", req.RepoID)
				err := wire.WriteStructPacket(stream, &wire.GetPackfileResponseHeader{ObjectIDs: []byte{}})
				if err != nil {
					log.Errorf("[packfile server] %+v", errors.WithStack(err))
					return
				}
				return
			}

			for _, id := range wire.UnflattenObjectIDs(req.ObjectIDs) {
				if r.HasObject(id) {
					availableObjectIDs = append(availableObjectIDs, id)
				}
			}

			err = wire.WriteStructPacket(stream, &wire.GetPackfileResponseHeader{ObjectIDs: wire.FlattenObjectIDs(availableObjectIDs)})
			if err != nil {
				log.Errorf("[packfile server] %+v", errors.WithStack(err))
				return
			}

			if len(availableObjectIDs) == 0 {
				continue
			}
		}

		// Write the packfile to the stream
		{
			log.Debugf("[packfile server] writing %v objects to packfile stream", len(availableObjectIDs))

			rPipe, wPipe := io.Pipe()

			go generatePackfile(wPipe, r, availableObjectIDs)
			err := pipePackfileAsPackets(rPipe, stream)
			if err != nil {
				return
			}
		}
	}
}

func generatePackfile(wPipe *io.PipeWriter, r *repo.Repo, availableObjectIDs [][]byte) {
	// @@TODO: to maximize packing efficiency, add commits first, then trees, then blobs.  Among
	// each object type, add most recent objects first.

	var err error
	defer func() { wPipe.CloseWithError(err) }()

	packbuilder, err := r.NewPackbuilder()
	if err != nil {
		log.Errorln("[packfile server] error instantiating packbuilder:", err)
		return
	}
	defer packbuilder.Free()

	for i := range availableObjectIDs {
		oid := util.OidFromBytes(availableObjectIDs[i])

		err = packbuilder.Insert(oid, "")
		if err != nil {
			log.Errorln("[packfile server] error adding object to packbuilder:", err)
			return
		}
	}

	err = packbuilder.Write(wPipe)
	if err != nil {
		log.Errorln("[packfile server] error writing packfile to stream:", err)
		return
	}
}

func pipePackfileAsPackets(rPipe io.Reader, stream io.Writer) error {
	data := make([]byte, OBJ_CHUNK_SIZE)
	for {
		n, err := io.ReadFull(rPipe, data)
		if err == io.EOF {
			break
		} else if err == io.ErrUnexpectedEOF {
			data = data[:n]
		} else if err != nil {
			log.Errorf("[packfile server] error reading packfile: %+v", err)
			return err
		}

		err = wire.WriteStructPacket(stream, &wire.GetPackfileResponsePacket{Data: data})
		if err != nil {
			log.Errorf("[packfile server] %+v", errors.WithStack(err))
			return err
		}
	}

	err := wire.WriteStructPacket(stream, &wire.GetPackfileResponsePacket{End: true})
	if err != nil {
		log.Errorf("[packfile server] %+v", errors.WithStack(err))
		return err
	}
	return nil
}
