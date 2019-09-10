package repo

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/Conscience/protocol/config"
	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/swarm/nodeevents"
	"github.com/Conscience/protocol/util"
)

type Manager struct {
	repos       map[string]*Repo
	reposByPath map[string]*Repo
	config      *config.Config
	eventBus    *nodeevents.EventBus
}

func NewManager(eventBus *nodeevents.EventBus, cfg *config.Config) (*Manager, error) {
	err := os.MkdirAll(cfg.Node.ReplicationRoot, os.ModePerm)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	rm := &Manager{
		repos:       map[string]*Repo{},
		reposByPath: map[string]*Repo{},
		config:      cfg,
		eventBus:    eventBus,
	}

	foundRepos := []string{}
	for _, path := range rm.config.Node.LocalRepos {
		_, err := rm.openRepo(path, false)
		if errors.Cause(err) == Err404 {
			log.Errorf("[repo manager] removing missing repo: %v", path)
			continue
		} else if err != nil {
			log.Errorf("[repo manager] error opening repo, removing: %v", err)
			continue
		}
		log.Infof("[repo manager] tracking local repo: %v", path)
		foundRepos = append(foundRepos, path)
	}

	if len(foundRepos) != len(rm.config.Node.LocalRepos) {
		err = rm.config.Update(func() error {
			rm.config.Node.LocalRepos = foundRepos
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return rm, nil
}

func (rm *Manager) EnsureLocalCheckoutExists(repoID string) (*Repo, error) {
	if r, known := rm.repos[repoID]; known {
		// @@TODO: test whether it physically exists on-disk?  and if not, recreate it?
		return r, nil
	}

	defaultPath := filepath.Join(rm.config.Node.ReplicationRoot, repoID)

	r, err := Open(defaultPath)
	if errors.Cause(err) == Err404 {
		r, err = Init(&InitOptions{RepoID: repoID, RepoRoot: defaultPath})
		if err != nil {
			return nil, err
		}

	} else if err != nil {
		return nil, err
	}

	rm.repos[repoID] = r

	err = r.SetupConfig(repoID)
	if err != nil {
		return nil, err
	}

	err = rm.config.Update(func() error {
		rm.config.Node.LocalRepos = util.StringSetAdd(rm.config.Node.LocalRepos, defaultPath)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (rm *Manager) TrackRepo(repoPath string, forceReload bool) (*Repo, error) {
	r, err := rm.openRepo(repoPath, forceReload)
	if r == nil {
		return nil, err
	}

	err = rm.config.Update(func() error {
		rm.config.Node.LocalRepos = util.StringSetAdd(rm.config.Node.LocalRepos, repoPath)
		return nil
	})
	if err != nil {
		return nil, err
	}

	repoID, err := r.RepoID()
	if err != nil {
		return nil, err
	}

	rm.eventBus.NotifyWatchers(nodeevents.MaybeEvent{
		EventType: nodeevents.EventType_AddedRepo,
		AddedRepoEvent: &nodeevents.AddedRepoEvent{
			RepoRoot: repoPath,
			RepoID:   repoID,
		},
	})

	return r, nil
}

func (rm *Manager) openRepo(repoPath string, forceReload bool) (*Repo, error) {
	if !forceReload {
		if r, exists := rm.reposByPath[repoPath]; exists {
			log.Warnf("[repo manager] already opened repo at path '%v' (doing nothing)", repoPath)
			return r, nil
		}
	}

	r, err := Open(repoPath)
	if err != nil {
		return nil, err
	}

	repoID, err := r.RepoID()
	if err != nil {
		return nil, err
	}

	rm.repos[repoID] = r
	rm.reposByPath[repoPath] = r

	return r, nil
}

func (rm *Manager) UntrackRepo(repoPath string) error {
	r, err := Open(repoPath)
	if err != nil {
		return err
	}

	repoID, err := r.RepoID()
	if err != nil {
		return err
	}

	delete(rm.repos, repoID)
	delete(rm.reposByPath, repoPath)

	return rm.removeLocalRepoFromConfig(repoPath)
}

func (rm *Manager) removeLocalRepoFromConfig(repoPath string) error {
	return rm.config.Update(func() error {
		rm.config.Node.LocalRepos = util.StringSetRemove(rm.config.Node.LocalRepos, repoPath)
		return nil
	})
}

func (rm *Manager) Repo(repoID string) *Repo {
	repo, ok := rm.repos[repoID]
	if !ok {
		return nil
	}
	return repo
}

func (rm *Manager) RepoAtPath(repoPath string) *Repo {
	repo, ok := rm.reposByPath[repoPath]
	if !ok {
		return nil
	}
	return repo
}

func (rm *Manager) RepoAtPathOrID(path string, repoID string) (*Repo, error) {
	if len(path) > 0 {
		r := rm.RepoAtPath(path)
		if r == nil {
			return nil, Err404
		} else {
			return r, nil
		}
	}
	if len(repoID) > 0 {
		r := rm.Repo(repoID)
		if r == nil {
			return nil, Err404
		} else {
			return r, nil
		}
	}

	return nil, errors.Errorf("must provide either 'path' or 'repoID'")
}

func (rm *Manager) RepoIDList() []string {
	repoIDs := make([]string, 0)
	for repoID := range rm.repos {
		repoIDs = append(repoIDs, repoID)
	}
	return repoIDs
}

func (rm *Manager) ForEachRepo(fn func(*Repo) error) error {
	for _, entry := range rm.reposByPath {
		err := fn(entry)
		if err != nil {
			return err
		}
	}
	return nil
}
