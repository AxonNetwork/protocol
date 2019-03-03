package p2pserver

import (
	"io"

	netp2p "github.com/libp2p/go-libp2p-net"
	"github.com/pkg/errors"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/repo"
	"github.com/Conscience/protocol/swarm/nodep2p"
	"github.com/Conscience/protocol/swarm/wire"
	"github.com/Conscience/protocol/util"
)

func (s *Server) HandlePackfileStreamRequest(stream netp2p.Stream) {
	defer stream.Close()

	for {
		req := wire.GetPackfileRequest{}
		err := wire.ReadStructPacket(stream, &req)
		if err != nil {
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
				return
			}
		}

		// Write the packfile to the stream
		{
			log.Debugf("[packfile server] writing %v objects to packfile stream", len(req.ObjectIDs)/20)

			rPipe, wPipe := io.Pipe()

			// Generate the packfile
			// @@TODO: to maximize packing efficiency, add commits first, then trees, then blobs.  Among
			// each object type, add most recent objects first.
			go func() {
				var err error
				defer func() {
					log.Errorf("[packfile server] packfile encoder: exiting with error: %v", err)
					wPipe.CloseWithError(err)
				}()

				log.Debugf("[packfile server] packfile encoder 11111: creating Packbuilder")
				packbuilder, err := r.NewPackbuilder()
				if err != nil {
					log.Errorln("[packfile server] error instantiating packbuilder:", err)
					return
				}
				log.Debugf("[packfile server] packfile encoder 22222: created Packbuilder")

				log.Debugf("[packfile server] packfile encoder 33333: %v availableObjectIDs", len(availableObjectIDs))
				for i := range availableObjectIDs {
					oid := util.OidFromBytes(availableObjectIDs[i])

					err = packbuilder.Insert(oid, "")
					if err != nil {
						log.Errorln("[packfile server] error adding object to packbuilder:", err)
						return
					}
				}
				log.Debugf("[packfile server] packfile encoder 44444: finished inserting oids.  writing packfile...")

				err = packbuilder.Write(wPipe)
				if err != nil {
					log.Errorln("[packfile server] error writing packfile to stream:", err)
					return
				}
				log.Debugf("[packfile server] packfile encoder 55555: done writing packfile")
			}()

			var totalWritten int
			for {
				log.Debugf("[packfile server] packfile: reading from encoder...")
				data := make([]byte, nodep2p.OBJ_CHUNK_SIZE)
				n, err := io.ReadFull(rPipe, data)
				if err == io.EOF {
					log.Debugf("[packfile server] packfile: EOF, done encoding")
					break
				} else if err == io.ErrUnexpectedEOF {
					log.Debugf("[packfile server] packfile: unexpected EOF (n = %v)", n)
					data = data[:n]
				} else if err != nil {
					log.Errorf("[packfile server] error reading packfile: %+v", err)
					return
				}
				log.Debugf("[packfile server] packfile: encoded %v bytes", n)

				err = wire.WriteStructPacket(stream, &wire.GetPackfileResponsePacket{Data: data})
				if err != nil {
					log.Errorf("[packfile server] %+v", errors.WithStack(err))
					return
				}
				totalWritten += n
				log.Debugln("[packfile server] packfile: bytes written:", totalWritten)
			}

			log.Debugln("[packfile server] packfile: done writing")
			err = wire.WriteStructPacket(stream, &wire.GetPackfileResponsePacket{End: true})
			if err != nil {
				log.Errorf("[packfile server] %+v", errors.WithStack(err))
				return
			}
		}
	}
}
