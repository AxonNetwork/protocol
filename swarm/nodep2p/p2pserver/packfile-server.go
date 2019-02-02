package p2pserver

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"

	netp2p "github.com/libp2p/go-libp2p-net"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/format/packfile"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/swarm/nodep2p"
	. "github.com/Conscience/protocol/swarm/wire"
)

func (s *Server) HandlePackfileStreamRequest(stream netp2p.Stream) {
	defer stream.Close()

	req := GetPackfileRequest{}
	err := ReadStructPacket(stream, &req)
	if err != nil {
		log.Errorf("[p2p server] %v", err)
		return
	}
	log.Debugf("[p2p server] incoming handshake")

	// Ensure the client has access
	{
		isAuth, err := s.isAuthorised(req.RepoID, req.Signature)
		if err != nil {
			log.Errorf("[p2p server] %v", err)
			return
		}

		if isAuth == false {
			err := WriteStructPacket(stream, &GetPackfileResponse{ErrUnauthorized: true})
			if err != nil {
				log.Errorf("[p2p server] %v", err)
				return
			}
			return
		}
	}

	// Write the packfile to the stream
	{
		log.Debugf("[p2p server] writing %v objects to packfile stream", len(req.ObjectIDs)/20)

		r := s.node.Repo(req.RepoID)
		if r == nil {
			log.Warnf("[p2p server] cannot find repo %v", req.RepoID)
			err := WriteStructPacket(stream, &GetPackfileResponse{ObjectIDs: []byte{}})
			if err != nil {
				log.Errorf("[p2p server] %v", err)
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
			log.Errorf("[p2p server] %v", err)
			return
		}

		if len(availableObjectIDs) == 0 {
			return
		}

		rPipe, wPipe := io.Pipe()

		go func() {
			defer wPipe.Close()
			cached := getCachedPackfile(availableObjectIDs)

			if cached != nil {
				defer cached.Close()

				_, err = io.Copy(wPipe, cached)
				if err != nil {
					wPipe.CloseWithError(err)
				}

			} else {
				cached = createCachedPackfile(availableObjectIDs)
				defer cached.Close()

				availableHashes := make([]gitplumbing.Hash, len(availableObjectIDs))
				for i := range availableObjectIDs {
					copy(availableHashes[i][:], availableObjectIDs[i])
				}

				mw := io.MultiWriter(wPipe, cached)
				enc := packfile.NewEncoder(mw, r.Storer, false)
				_, err = enc.Encode(availableHashes, 10) // @@TODO: do we need to negotiate the packfile window with the client?
				if err != nil {
					wPipe.CloseWithError(err)
				}
			}
		}()

		end := false
		for {
			data := make([]byte, nodep2p.OBJ_CHUNK_SIZE)
			n, err := io.ReadFull(rPipe, data)
			if err == io.EOF {
				end = true
			} else if err == io.ErrUnexpectedEOF {
				data = data[:n]
				end = true
			} else if err != nil {
				log.Errorln("[p2p server] error reading packfile")
			}

			err = WriteStructPacket(stream, &GetPackfileResponsePacket{
				End:    end,
				Length: n,
			})
			if err != nil {
				log.Errorf("[p2p server] %v", err)
				return
			}

			n, err = stream.Write(data)
			if err != nil {
				log.Errorf("[p2p server] %v", err)
				return
			}

		}

		_, err := io.Copy(stream, rPipe)
		if err != nil {
			log.Errorln("[p2p server] error sending packfile:", err)
		}
	}
}

func getCachedPackfile(objectIDs [][]byte) *os.File {
	sorted := SortByteSlices(objectIDs)
	flattened := FlattenObjectIDs(sorted)
	hash := sha256.Sum256(flattened)
	hex := fmt.Sprintf("%0x", hash[:])

	cacheDir := filepath.Join(os.TempDir(), "conscience-packfile-cache")
	err := os.MkdirAll(cacheDir, os.ModePerm)
	if err != nil {
		log.Errorln("[p2p server] getCachedPackfile:", err)
		return nil
	}

	f, err := os.Open(filepath.Join(cacheDir, hex))
	if os.IsNotExist(err) {
		return nil
	}
	return f
}

func createCachedPackfile(objectIDs [][]byte) *os.File {
	sorted := SortByteSlices(objectIDs)
	flattened := FlattenObjectIDs(sorted)
	hash := sha256.Sum256(flattened)
	hex := fmt.Sprintf("%0x", hash[:])

	cacheDir := filepath.Join(os.TempDir(), "conscience-packfile-cache")
	err := os.MkdirAll(cacheDir, os.ModePerm)
	if err != nil {
		log.Errorln("[p2p server] createCachedPackfile:", err)
		return nil
	}

	log.Infoln("[p2p server] caching packfile", filepath.Join(cacheDir, hex))
	f, err := os.Create(filepath.Join(cacheDir, hex))
	if os.IsNotExist(err) {
		return nil
	}
	return f
}
