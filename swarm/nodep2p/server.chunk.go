package nodep2p

import (
	"encoding/hex"
	"io"
	"os"
	"path/filepath"

	netp2p "github.com/libp2p/go-libp2p-net"
	"github.com/pkg/errors"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/repo"
)

func (h *Host) handleChunkStreamRequest(stream netp2p.Stream) {
	defer stream.Close()

	for {
		var req GetChunkRequest
		err := ReadMsg(stream, &req)
		if err == io.EOF {
			log.Debugf("[packfile server] peer closed stream")
			return
		} else if err != nil {
			log.Errorf("[chunk server] %+v", errors.WithStack(err))
			return
		}
		log.Debugf("[chunk server] incoming handshake")

		//
		// Ensure the client has access, then send a header packet with the chunk's length
		//
		var chunkPath string
		{
			isAuth, err := h.isAuthorised(req.RepoID, req.Signature)
			if err != nil {
				log.Errorf("[chunk server] %+v", errors.WithStack(err))
				return
			}

			if isAuth == false {
				err := WriteMsg(stream, &GetChunkResponseHeader{ErrUnauthorized: true})
				if err != nil {
					log.Errorf("[chunk server] %+v", errors.WithStack(err))
					return
				}
				return
			}

			r := h.repoManager.Repo(req.RepoID)
			chunkStr := hex.EncodeToString(req.ChunkID)
			chunkPath = filepath.Join(r.Path(), ".git", repo.CONSCIENCE_DATA_SUBDIR, chunkStr)

			stat, err := os.Stat(chunkPath)
			if err != nil {
				err = WriteMsg(stream, &GetChunkResponseHeader{ErrObjectNotFound: true})
				if err != nil {
					log.Errorf("[chunk server] %+v", errors.WithStack(err))
					break
				} else {
					continue
				}
			}

			err = WriteMsg(stream, &GetChunkResponseHeader{Length: stat.Size()})
			if err != nil {
				log.Errorf("[chunk server] %+v", errors.WithStack(err))
				return
			}
		}

		//
		// Send the chunk to the client
		//
		{
			log.Infof("[chunk server] writing chunk %0x", req.ChunkID)

			chunkFile, err := os.Open(chunkPath)
			if err != nil {
				log.Errorf("[chunk server] %+v", errors.WithStack(err))
				break
			}

			for {
				data := make([]byte, OBJ_CHUNK_SIZE)
				n, err := io.ReadFull(chunkFile, data)
				if err == io.EOF {
					break

				} else if err == io.ErrUnexpectedEOF {
					data = data[:n]

				} else if err != nil {
					log.Errorf("[chunk server] %+v", err)
					return
				}

				err = WriteMsg(stream, &GetChunkResponsePacket{Data: data})
				if err != nil {
					log.Errorf("[chunk server] %+v", errors.WithStack(err))
					return
				}
			}

			err = WriteMsg(stream, &GetChunkResponsePacket{End: true})
			if err != nil {
				log.Errorf("[packfile server] %+v", errors.WithStack(err))
				return
			}
		}
	}
}
