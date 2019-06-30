package nodep2p

import (
	"context"
	"io"
	"time"

	netp2p "github.com/libp2p/go-libp2p-net"
	"github.com/pkg/errors"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/repo"
	"github.com/Conscience/protocol/util"
)

func (h *Host) handlePackfileStreamRequest(stream netp2p.Stream) {
	defer stream.Close()

	var err error
	defer func() {
		if err != nil {
			log.Errorf("[packfile server] %+v", errors.WithStack(err))
		}
	}()

	for {
		var req GetPackfileRequest
		err = ReadMsg(stream, &req)
		if err == io.EOF {
			return
		} else if err != nil {
			return
		}
		log.Debugf("[packfile server] incoming handshake")

		// Ensure the client has access, then report which of the objectIDs we can provide.
		var availableObjectIDs [][]byte
		var r *repo.Repo
		{
			addr, err := h.ethClient.AddrFromSignedHash([]byte(req.RepoID), req.Signature)
			if err != nil {
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			isAuth, err := h.ethClient.AddressHasPullAccess(ctx, addr, req.RepoID)
			if err != nil {
				return
			}

			if isAuth == false {
				log.Warnf("[p2p server] address 0x%0x does not have pull access", addr.Bytes())

				err = WriteMsg(stream, &GetPackfileResponseHeader{ErrUnauthorized: true})
				return
			}

			r = h.repoManager.Repo(req.RepoID)
			if r == nil {
				log.Warnf("[packfile server] cannot find repo %v", req.RepoID)

				err = WriteMsg(stream, &GetPackfileResponseHeader{ObjectIDs: []byte{}})
				return
			}

			for _, id := range UnflattenObjectIDs(req.ObjectIDs) {
				if r.HasObject(id) {
					availableObjectIDs = append(availableObjectIDs, id)
				}
			}

			err = WriteMsg(stream, &GetPackfileResponseHeader{ObjectIDs: FlattenObjectIDs(availableObjectIDs)})
			if err != nil {
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
			err = pipePackfileAsPackets(rPipe, stream)
		}
	}
}

func generatePackfile(wPipe *io.PipeWriter, r *repo.Repo, availableObjectIDs [][]byte) {
	// @@TODO: to maximize packing efficiency, add commits first, then trees, then blobs.  Among
	// each object type, add most recent objects first.

	var err error
	defer func() { wPipe.CloseWithError(errors.WithStack(err)) }()

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

func pipePackfileAsPackets(rPipe io.Reader, stream io.Writer) (err error) {
	defer func() { err = errors.WithStack(err) }()

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

		err = WriteMsg(stream, &GetPackfileResponsePacket{Data: data})
		if err != nil {
			return err
		}
	}

	err = WriteMsg(stream, &GetPackfileResponsePacket{End: true})
	if err != nil {
		return err
	}
	return nil
}
