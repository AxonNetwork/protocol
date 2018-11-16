package strategy

import (
	"context"
	"io"
	"sync"
	"time"

	peer "gx/ipfs/QmdVrMn1LhB4ybb8hMVaMLXnA8XRSewMnK6YqXKXoTcRvN/go-libp2p-peer"

	"github.com/pkg/errors"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"

	"github.com/Conscience/protocol/config"
	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/repo"
	. "github.com/Conscience/protocol/swarm/wire"
	"github.com/Conscience/protocol/util"
)

type SmartClient struct {
	inflightLimiter chan struct{}
	node            INode
	repo            *repo.Repo
	config          *config.Config
	flatHead        []byte
	flatHistory     []byte
	jobQueue        chan job
}

type job struct {
	objectID    []byte
	failedPeers map[peer.ID]bool
}

func NewSmartClient(node INode, repo *repo.Repo, config *config.Config) *SmartClient {
	sc := &SmartClient{
		inflightLimiter: make(chan struct{}, 5),
		node:            node,
		repo:            repo,
		config:          config,
		jobQueue:        make(chan job),
	}
	for i := 0; i < 5; i++ {
		sc.inflightLimiter <- struct{}{}
	}
	return sc
}

func (sc *SmartClient) FetchFromCommit(ctx context.Context, repoID string, commit string) <-chan MaybeChunk {
	ch := make(chan MaybeChunk)
	wg := &sync.WaitGroup{}

	// Request the manifest and load the job queue up with everything it contains
	go func() {
		defer close(ch)
		defer close(sc.jobQueue)

		flatHead, flatHistory, err := sc.requestManifestFromSwarm(ctx, repoID, commit)
		if err != nil {
			ch <- MaybeChunk{Error: err}
			return
		}

		allObjects := append(flatHead, flatHistory...)

		for i := 0; i < len(allObjects); i += 20 {
			wg.Add(1)
			sc.jobQueue <- job{allObjects[i : i+20], make(map[peer.ID]bool)}
		}

		wg.Wait()
	}()

	// Consume the job queue with connections managed by a peerPool{}
	go func() {
		p, err := newPeerPool(ctx, sc.node, repoID, 4)
		if err != nil {
			ch <- MaybeChunk{Error: err}
			return
		}

		for j := range sc.jobQueue {
			conn := p.GetConn()
			if conn == IPeerConnection(nil) {
				log.Errorln("[smart client] nil PeerConnection, operation canceled?")
				return
			}

			go sc.fetchObject(ctx, p, conn, j, wg, ch)
		}
	}()

	return ch
}

func (sc *SmartClient) requestManifestFromSwarm(ctx context.Context, repoID string, commit string) ([]byte, []byte, error) {
	c, err := util.CidForString(repoID)
	if err != nil {
		return nil, nil, err
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(sc.config.Node.FindProviderTimeout))
	defer cancel()

	for provider := range sc.node.FindProvidersAsync(ctxTimeout, c, 10) {
		if provider.ID != sc.node.ID() {
			// We found a peer with the object
			head, history, err := sc.requestManifestFromPeer(ctx, provider.ID, repoID, commit)
			if err != nil {
				log.Errorln("[requestManifestFromSwarm]", err)
				continue
			}
			return head, history, nil
		}
	}
	return nil, nil, errors.Errorf("could not find provider for %v : %v", repoID, commit)
}

func (sc *SmartClient) requestManifestFromPeer(ctx context.Context, peerID peer.ID, repoID string, commit string) ([]byte, []byte, error) {
	log.Debugf("[p2p object client] requesting manifest %v/%v from peer %v", repoID, commit, peerID.Pretty())

	// Open the stream
	stream, err := sc.node.NewStream(ctx, peerID, MANIFEST_PROTO)
	if err != nil {
		return nil, nil, err
	}

	sig, err := sc.node.SignHash([]byte(commit))
	if err != nil {
		return nil, nil, err
	}

	// Write the request packet to the stream
	err = WriteStructPacket(stream, &GetManifestRequest{RepoID: repoID, Commit: commit, Signature: sig})
	if err != nil {
		return nil, nil, err
	}

	// // Read the response
	resp := GetManifestResponse{}
	err = ReadStructPacket(stream, &resp)
	if err != nil {
		return nil, nil, err
	} else if !resp.Authorized {
		return nil, nil, errors.Wrapf(ErrUnauthorized, "%v:%0x", repoID, commit)
	} else if !resp.HasCommit {
		return nil, nil, errors.Wrapf(ErrObjectNotFound, "%v:%0x", repoID, commit)
	}

	log.Debugf("[p2p object client] got manifest metadata %+v", resp)

	flatHead := make([]byte, resp.HeadLen)
	_, err = io.ReadFull(stream, flatHead)
	if err != nil {
		return nil, nil, err
	}

	flatHistory := make([]byte, resp.HistoryLen)
	_, err = io.ReadFull(stream, flatHistory)
	if err != nil {
		return nil, nil, err
	}

	return flatHead, flatHistory, nil
}

func (sc *SmartClient) fetchObject(ctx context.Context, p *peerPool, conn IPeerConnection, j job, wg *sync.WaitGroup, ch chan MaybeChunk) {
	var err error

	defer func() {
		if err == nil {
			wg.Done()
			p.ReturnConn(conn, false)
		} else {
			// @@TODO: mark failed peer on job{}
			sc.jobQueue <- j
			// @@TODO: maybe call ReturnConn with true if the peer should be discarded
			p.ReturnConn(conn, false)
		}
	}()

	if sc.repo != nil && sc.repo.HasObject(j.objectID) {
		return
	}

	objReader, err := conn.RequestObject(ctx, j.objectID)
	if err != nil {
		ch <- MaybeChunk{Error: err}
		return
	}
	defer objReader.Close()

	var hash [20]byte
	copy(hash[:], j.objectID)

	// if object has no data, still need to send to channel
	if objReader.Len() == 0 {
		ch <- MaybeChunk{
			ObjHash: gitplumbing.Hash(hash),
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
			ObjHash: gitplumbing.Hash(hash),
			ObjType: objReader.Type(),
			ObjLen:  objReader.Len(),
			Data:    data,
		}
	}
}
