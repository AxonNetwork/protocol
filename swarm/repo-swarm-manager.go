package swarm

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/Conscience/protocol/repo"
	"github.com/Conscience/protocol/util"
	"github.com/pkg/errors"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
)

const OBJ_CHUNK_SIZE = 16384 //16kb

type RepoSwarmManager struct {
	inflightLimiter chan struct{}
	node            *Node
	repo            *repo.Repo
	flatHead        []byte
	flatHistory     []byte
}

func NewRepoSwarmManager(node *Node, repo *repo.Repo) *RepoSwarmManager {
	sm := &RepoSwarmManager{
		inflightLimiter: make(chan struct{}, 5),
		node:            node,
		repo:            repo,
	}
	for i := 0; i < 5; i++ {
		sm.inflightLimiter <- struct{}{}
	}
	return sm
}

type MaybeChunk struct {
	ObjHash gitplumbing.Hash
	ObjType gitplumbing.ObjectType
	ObjLen  uint64
	Data    []byte
	Error   error
}

func (sm *RepoSwarmManager) FetchFromCommit(ctx context.Context, repoID string, commit string) <-chan MaybeChunk {
	ch := make(chan MaybeChunk)
	flatHead, flatHistory, err := sm.requestManifest(ctx, repoID, commit)
	if err != nil {
		go func() { ch <- MaybeChunk{Error: err} }()
		return ch
	}
	allObjects := append(flatHead, flatHistory...)
	go sm.fetchObjects(repoID, allObjects, ch)
	return ch
}

func (sm *RepoSwarmManager) requestManifest(ctx context.Context, repoID string, commit string) ([]byte, []byte, error) {
	c, err := cidForString(repoID)
	if err != nil {
		return nil, nil, err
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(sm.node.Config.Node.FindProviderTimeout))
	defer cancel()

	for provider := range sm.node.dht.FindProvidersAsync(ctxTimeout, c, 10) {
		if provider.ID != sm.node.host.ID() {
			// We found a peer with the object
			return sm.node.RequestManifest(ctx, provider.ID, repoID, commit)
		}
	}
	return nil, nil, errors.Errorf("could not find provider for %v : %v", repoID, commit)
}

func (sm *RepoSwarmManager) fetchObjects(repoID string, objects []byte, ch chan MaybeChunk) {
	wg := &sync.WaitGroup{}
	for i := 0; i < len(objects); i += 20 {
		hash := [20]byte{}
		copy(hash[:], objects[i:i+20])
		wg.Add(1)
		go sm.fetchObject(repoID, gitplumbing.Hash(hash), wg, ch)
	}
	wg.Wait()
	close(ch)
}

func (sm *RepoSwarmManager) fetchObject(repoID string, hash gitplumbing.Hash, wg *sync.WaitGroup, ch chan MaybeChunk) {
	defer wg.Done()
	if sm.repo != nil && sm.repo.HasObject(hash[:]) {
		return
	}
	objReader, err := sm.fetchObjStream(repoID, hash)
	if err != nil {
		ch <- MaybeChunk{Error: err}
		return
	}
	defer objReader.Close()
	for {
		data := make([]byte, OBJ_CHUNK_SIZE)
		n, err := io.ReadFull(objReader.Reader, data)
		if err == io.EOF {
			// read no bytes
			break
		} else if err == io.ErrUnexpectedEOF {
			data = data[:n]
			break

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

func (sm *RepoSwarmManager) fetchObjStream(repoID string, hash gitplumbing.Hash) (*util.ObjectReader, error) {
	<-sm.inflightLimiter
	defer func() { sm.inflightLimiter <- struct{}{} }()

	// Fetch an object stream from the node via RPC
	// @@TODO: give context a timeout and make it configurable
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return sm.node.GetObjectReader(ctx, repoID, hash[:])
}
