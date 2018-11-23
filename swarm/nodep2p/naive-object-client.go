package nodep2p

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-peer"
	"github.com/pkg/errors"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"

	"github.com/Conscience/protocol/config"
	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/repo"
	. "github.com/Conscience/protocol/swarm/wire"
	"github.com/Conscience/protocol/util"
)

type NaiveClient struct {
	inflightLimiter chan struct{}
	node            INode
	repo            *repo.Repo
	config          *config.Config
	flatHead        []byte
	flatHistory     []byte
}

func NewNaiveClient(node INode, repo *repo.Repo, config *config.Config) *NaiveClient {
	nc := &NaiveClient{
		inflightLimiter: make(chan struct{}, 5),
		node:            node,
		repo:            repo,
		config:          config,
	}
	for i := 0; i < 5; i++ {
		nc.inflightLimiter <- struct{}{}
	}
	return nc
}

func (nc *NaiveClient) FetchFromCommit(ctx context.Context, repoID string, commit string) <-chan MaybeChunk {
	ch := make(chan MaybeChunk)

	go func() {
		defer close(ch)

		allObjects, err := nc.requestManifestFromSwarm(ctx, repoID, commit)
		if err != nil {
			ch <- MaybeChunk{Error: err}
			return
		}

		wg := &sync.WaitGroup{}

		for i := 0; i < len(allObjects); i += 20 {
			var hash [20]byte
			copy(hash[:], allObjects[i:i+20])

			wg.Add(1)
			go nc.fetchObject(repoID, gitplumbing.Hash(hash), wg, ch)
		}
		wg.Wait()
	}()

	return ch
}

func (nc *NaiveClient) requestManifestFromSwarm(ctx context.Context, repoID string, commit string) ([]byte, error) {
	c, err := util.CidForString(repoID)
	if err != nil {
		return nil, err
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(nc.config.Node.FindProviderTimeout))
	defer cancel()

	for provider := range nc.node.FindProvidersAsync(ctxTimeout, c, 10) {
		if provider.ID != nc.node.ID() {
			// We found a peer with the object
			allObjects, err := nc.requestManifestFromPeer(ctx, provider.ID, repoID, commit)
			if err != nil {
				log.Errorln("[requestManifestFromSwarm]", err)
				continue
			}
			return allObjects, nil
		}
	}
	return nil, errors.Errorf("could not find provider for %v : %v", repoID, commit)
}

func (nc *NaiveClient) requestManifestFromPeer(ctx context.Context, peerID peer.ID, repoID string, commit string) ([]byte, error) {
	log.Debugf("[p2p object client] requesting manifest %v/%v from peer %v", repoID, commit, peerID.Pretty())

	// Open the stream
	stream, err := nc.node.NewStream(ctx, peerID, MANIFEST_PROTO)
	if err != nil {
		return nil, err
	}

	sig, err := nc.node.SignHash([]byte(commit))
	if err != nil {
		return nil, err
	}

	// Write the request packet to the stream
	err = WriteStructPacket(stream, &GetManifestRequest{RepoID: repoID, Commit: commit, Signature: sig})
	if err != nil {
		return nil, err
	}

	// // Read the response
	resp := GetManifestResponse{}
	err = ReadStructPacket(stream, &resp)
	if err != nil {
		return nil, err
	} else if !resp.Authorized {
		return nil, errors.Wrapf(ErrUnauthorized, "%v:%0x", repoID, commit)
	} else if !resp.HasCommit {
		return nil, errors.Wrapf(ErrObjectNotFound, "%v:%0x", repoID, commit)
	}

	log.Debugf("[p2p object client] got manifest metadata %+v", resp)

	allObjects := make([]byte, 0)
	for {
		obj := ManifestObject{}
		err = ReadStructPacket(stream, &obj)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		allObjects = append(allObjects, obj.Hash...)
	}

	return allObjects, nil
}

func (nc *NaiveClient) fetchObject(repoID string, hash gitplumbing.Hash, wg *sync.WaitGroup, ch chan MaybeChunk) {
	defer wg.Done()

	if nc.repo != nil && nc.repo.HasObject(hash[:]) {
		return
	}

	objReader, err := nc.fetchObjectStream(repoID, hash)
	if err != nil {
		ch <- MaybeChunk{Error: err}
		return
	}
	defer objReader.Close()

	// if object has no data, still need to send to channel
	if objReader.Len() == 0 {
		ch <- MaybeChunk{
			ObjHash: hash,
			ObjType: objReader.Type(),
			ObjLen:  objReader.Len(),
			Data:    make([]byte, 0),
		}
		return
	}

	for {
		data := make([]byte, OBJ_CHUNK_SIZE)
		n, err := io.ReadFull(objReader.Reader, data)
		if err == io.EOF {
			// read no bytes
			break
		} else if err == io.ErrUnexpectedEOF {
			data = data[:n]
		} else if err != nil {
			ch <- MaybeChunk{Error: err}
			break
		}
		ch <- MaybeChunk{
			ObjHash: hash,
			ObjType: objReader.Type(),
			ObjLen:  objReader.Len(),
			Data:    data,
		}
	}
}

// Attempts to open a stream to the given object.  If we have it locally, the object is read from
// the filesystem.  Otherwise, we look for a peer and stream it over a p2p connection.
func (nc *NaiveClient) fetchObjectStream(repoID string, hash gitplumbing.Hash) (*util.ObjectReader, error) {
	<-nc.inflightLimiter
	defer func() { nc.inflightLimiter <- struct{}{} }()

	// Fetch an object stream from the node via RPC
	// @@TODO: give context a timeout and make it configurable
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID := hash[:]

	// r := n.RepoManager.Repo(repoID)

	// // If we detect that we already have the object locally, just open a regular file stream
	// if r != nil && r.HasObject(objectID) {
	//  return r.OpenObject(objectID)
	// }

	c, err := util.CidForString(repoID)
	if err != nil {
		return nil, err
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(nc.config.Node.FindProviderTimeout))
	defer cancel()

	for provider := range nc.node.FindProvidersAsync(ctxTimeout, c, 10) {
		if provider.ID != nc.node.ID() {
			// We found a peer with the object
			ctxRequestObject, _ := context.WithTimeout(ctx, 15*time.Second)
			objectReader, err := nc.fetchObjectStreamFromPeer(ctxRequestObject, provider.ID, repoID, objectID)
			if err != nil {
				log.Warnln("[p2p object client] error requesting object:", err)
				continue
			}
			return objectReader, nil
		}
	}
	return nil, errors.Errorf("could not find provider for %v : %0x", repoID, objectID)
}

// Opens an outgoing request to another Node for the given object.
func (nc *NaiveClient) fetchObjectStreamFromPeer(ctx context.Context, peerID peer.ID, repoID string, objectID []byte) (*util.ObjectReader, error) {
	log.Debugf("[p2p object client] requesting object %v/%0x from peer %v", repoID, objectID, peerID.Pretty())

	// Open the stream
	stream, err := nc.node.NewStream(ctx, peerID, OBJECT_PROTO)
	if err != nil {
		return nil, err
	}

	sig, err := nc.node.SignHash(objectID)
	if err != nil {
		return nil, err
	}

	// Write the request packet to the stream
	err = WriteStructPacket(stream, &GetObjectRequestSigned{RepoID: repoID, ObjectID: objectID, Signature: sig})
	if err != nil {
		return nil, err
	}

	// Read the response
	resp := GetObjectResponse{}
	err = ReadStructPacket(stream, &resp)
	if err != nil {
		return nil, err
	} else if resp.Unauthorized {
		return nil, errors.Wrapf(ErrUnauthorized, "%v:%0x", repoID, objectID)
	} else if !resp.HasObject {
		return nil, errors.Wrapf(ErrObjectNotFound, "%v:%0x", repoID, objectID)
	}

	log.Debugf("[p2p object client] got object metadata %+v", resp)

	or := &util.ObjectReader{
		Reader:     stream,
		Closer:     stream,
		ObjectType: resp.ObjectType,
		ObjectLen:  resp.ObjectLen,
	}
	return or, nil
}
