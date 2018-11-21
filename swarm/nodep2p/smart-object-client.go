package nodep2p

// import (
// 	"context"
// 	"encoding/hex"
// 	"io"
// 	"sync"
// 	"time"

// 	peer "github.com/libp2p/go-libp2p-peer"
// 	"github.com/pkg/errors"
// 	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"

// 	"github.com/Conscience/protocol/config"
// 	"github.com/Conscience/protocol/log"
// 	"github.com/Conscience/protocol/repo"
// 	. "github.com/Conscience/protocol/swarm/wire"
// 	"github.com/Conscience/protocol/util"
// )

// type SmartClient struct {
// 	node        INode
// 	repo        *repo.Repo
// 	config      *config.Config
// 	flatHead    []byte
// 	flatHistory []byte
// }

// type job struct {
// 	objectID    []byte
// 	failedPeers map[peer.ID]bool
// }

// var ErrFetchingFromPeer = errors.New("fetching from peer")

// func NewSmartClient(node INode, repo *repo.Repo, config *config.Config) *SmartClient {
// 	sc := &SmartClient{
// 		node:   node,
// 		repo:   repo,
// 		config: config,
// 	}
// 	return sc
// }

// func (sc *SmartClient) FetchFromCommit(ctx context.Context, repoID string, commit string) <-chan MaybeChunk {
// 	ch := make(chan MaybeChunk)
// 	wg := &sync.WaitGroup{}

// 	flatHead, flatHistory, err := sc.requestManifestFromSwarm(ctx, repoID, commit)
// 	if err != nil {
// 		go func() {
// 			defer close(ch)
// 			ch <- MaybeChunk{Error: err}
// 		}()
// 		return ch
// 	}

// 	allObjects := append(flatHead, flatHistory...)
// 	numObjects := len(allObjects) / 20
// 	jobQueue := make(chan job, numObjects)

// 	// Load the job queue up with everything in the manifest
// 	go func() {
// 		defer close(ch)
// 		defer close(jobQueue)

// 		for i := 0; i < len(allObjects); i += 20 {
// 			wg.Add(1)
// 			jobQueue <- job{allObjects[i : i+20], make(map[peer.ID]bool)}
// 		}

// 		wg.Wait()
// 	}()

// 	// Consume the job queue with connections managed by a peerPool{}
// 	go func() {
// 		pool, err := newPeerPool(ctx, sc.node, repoID, 4)
// 		if err != nil {
// 			ch <- MaybeChunk{Error: err}
// 			return
// 		}
// 		defer pool.Close()

// 		for j := range jobQueue {
// 			conn := pool.GetConn()
// 			if conn == nil {
// 				log.Errorln("[smart client] nil PeerConnection, operation canceled?")
// 				return
// 			}

// 			go func(j job) {
// 				err := sc.fetchObject(ctx, conn, j, ch)
// 				if err != nil {
// 					log.Errorln("[smart client] fetchObject:", err)
// 					if errors.Cause(err) == ErrFetchingFromPeer {
// 						// @@TODO: mark failed peer on job{}
// 						// @@TODO: maybe call ReturnConn with true if the peer should be discarded
// 					}
// 					jobQueue <- j
// 					pool.ReturnConn(conn, false)

// 				} else {
// 					wg.Done()
// 					pool.ReturnConn(conn, false)
// 				}
// 			}(j)
// 		}
// 	}()

// 	return ch
// }

// func (sc *SmartClient) requestManifestFromSwarm(ctx context.Context, repoID string, commit string) ([]byte, []byte, error) {
// 	c, err := util.CidForString(repoID)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(sc.config.Node.FindProviderTimeout))
// 	defer cancel()

// 	for provider := range sc.node.FindProvidersAsync(ctxTimeout, c, 10) {
// 		if provider.ID != sc.node.ID() {
// 			// We found a peer with the object
// 			head, history, err := sc.requestManifestFromPeer(ctx, provider.ID, repoID, commit)
// 			if err != nil {
// 				log.Errorln("[smart client] requestManifestFromPeer:", err)
// 				continue
// 			}
// 			return head, history, nil
// 		}
// 	}
// 	return nil, nil, errors.Errorf("could not find provider for repo '%v'", repoID)
// }

// func (sc *SmartClient) requestManifestFromPeer(ctx context.Context, peerID peer.ID, repoID string, commit string) ([]byte, []byte, error) {
// 	log.Debugf("[p2p object client] requesting manifest %v/%v from peer %v", repoID, commit, peerID.Pretty())

// 	// Open the stream
// 	stream, err := sc.node.NewStream(ctx, peerID, MANIFEST_PROTO)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	sig, err := sc.node.SignHash([]byte(commit))
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	// Write the request packet to the stream
// 	err = WriteStructPacket(stream, &GetManifestRequest{RepoID: repoID, Commit: commit, Signature: sig})
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	// // Read the response
// 	resp := GetManifestResponse{}
// 	err = ReadStructPacket(stream, &resp)
// 	if err != nil {
// 		return nil, nil, err
// 	} else if !resp.Authorized {
// 		return nil, nil, errors.Wrapf(ErrUnauthorized, "%v:%0x", repoID, commit)
// 	} else if !resp.HasCommit {
// 		return nil, nil, errors.Wrapf(ErrObjectNotFound, "%v:%0x", repoID, commit)
// 	}

// 	log.Debugf("[p2p object client] got manifest metadata %+v", resp)

// 	flatHead := make([]byte, resp.HeadLen)
// 	_, err = io.ReadFull(stream, flatHead)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	flatHistory := make([]byte, resp.HistoryLen)
// 	_, err = io.ReadFull(stream, flatHistory)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	return flatHead, flatHistory, nil
// }

// func (sc *SmartClient) fetchObject(ctx context.Context, conn *PeerConnection, j job, ch chan MaybeChunk) error {
// 	if sc.repo != nil && sc.repo.HasObject(j.objectID) {
// 		return nil
// 	}

// 	objReader, err := conn.RequestObject(ctx, j.objectID)
// 	if err != nil {
// 		err := errors.Wrapf(ErrFetchingFromPeer, "tried requesting %v from peer %v: %v", hex.EncodeToString(j.objectID), conn.peerID, err)
// 		log.Errorf("[smart client]", err)
// 		return err
// 	}
// 	defer objReader.Close()

// 	var hash [20]byte
// 	copy(hash[:], j.objectID)

// 	// if object has no data, still need to send to channel
// 	if objReader.Len() == 0 {
// 		ch <- MaybeChunk{
// 			ObjHash: gitplumbing.Hash(hash),
// 			ObjType: objReader.Type(),
// 			ObjLen:  objReader.Len(),
// 			Data:    make([]byte, 0),
// 		}
// 		return nil
// 	}

// 	data := make([]byte, OBJ_CHUNK_SIZE)
// 	for {
// 		n, err := io.ReadFull(objReader.Reader, data)
// 		if err == io.EOF {
// 			// read no bytes
// 			break
// 		} else if err == io.ErrUnexpectedEOF {
// 			data = data[:n]
// 		} else if err != nil {
// 			return err
// 		}
// 		ch <- MaybeChunk{
// 			ObjHash: gitplumbing.Hash(hash),
// 			ObjType: objReader.Type(),
// 			ObjLen:  objReader.Len(),
// 			Data:    data,
// 		}
// 	}

// 	return nil
// }
