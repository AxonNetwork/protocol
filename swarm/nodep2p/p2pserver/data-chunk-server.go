package p2pserver

import (
	"encoding/hex"
	"io"
	"os"
	"path/filepath"

	netp2p "github.com/libp2p/go-libp2p-net"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/repo"
	. "github.com/Conscience/protocol/swarm/wire"
)

func (s *Server) HandleDataChunkStream(stream netp2p.Stream, repoID string) {
	defer stream.Close()

	log.Infof("[chunk server] data chunk stream open")

	for {
		req := GetChunkRequest{}
		err := ReadStructPacket(stream, &req)
		if err == io.EOF {
			log.Errorf("[chunk server] peer closed stream")
			break
		} else if err != nil {
			log.Errorf("[chunk server] %v", err)
			break
		}

		r := s.node.Repo(repoID)
		chunkStr := hex.EncodeToString(req.ChunkID)
		p := filepath.Join(r.Path, ".git", repo.CONSCIENCE_DATA_SUBDIR, chunkStr)

		stat, err := os.Stat(p)
		if err != nil {
			err = WriteStructPacket(stream, &GetChunkResponse{ErrObjectNotFound: true})
			if err != nil {
				log.Errorf("[chunk server] %v", err)
				break
			} else {
				continue
			}
		}

		log.Infof("[chunk server] writing chunk %v", chunkStr)

		length := stat.Size()
		err = WriteStructPacket(stream, &GetChunkResponse{Length: int(length)})
		if err != nil {
			log.Errorf("[chunk server] %v", err)
			break
		}

		f, err := os.Open(p)
		if err != nil {
			log.Errorf("[chunk server] %v", err)
			break
		}

		n, err := io.Copy(stream, f)
		if err != nil {
			log.Errorf("[chunk server] %v", err)
			break
		} else if n < length {
			log.Errorf("[chunk server] did not send full file")
			break
		}
	}
}
