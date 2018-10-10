package swarm

import (
	"path/filepath"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/src-d/go-git.v4"

	"github.com/Conscience/protocol/config"
	"github.com/Conscience/protocol/repo"
	"github.com/Conscience/protocol/util"
)

type RepoManager struct {
	repos       map[string]*repo.Repo
	reposByPath map[string]*repo.Repo
	config      *config.Config
}

func NewRepoManager(config *config.Config) *RepoManager {
	rm := &RepoManager{
		repos:       map[string]*repo.Repo{},
		reposByPath: map[string]*repo.Repo{},
		config:      config,
	}

	foundRepos := []string{}
	for _, path := range rm.config.Node.LocalRepos {
		_, err := rm.openRepo(path)
		if errors.Cause(err) == git.ErrRepositoryNotExists {
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
		rm.config.Update(func() error {
			rm.config.Node.LocalRepos = foundRepos
			return nil
		})
	}

	// for _, repoID := range rm.config.Node.ReplicateRepos {
	//     _, err := rm.EnsureLocalCheckoutExists(repoID)
	//     if err != nil {
	//         log.Errorf("[repo manager] %v", err)
	//         continue
	//     }
	// }

	return rm
}

func (rm *RepoManager) EnsureLocalCheckoutExists(repoID string) (*repo.Repo, error) {
	if r, known := rm.repos[repoID]; known {
		// @@TODO: test whether it physically exists on-disk?  and if not, recreate it?
		return r, nil
	}

	defaultPath := filepath.Join(rm.config.Node.ReplicationRoot, repoID)

	r, err := repo.EnsureExists(defaultPath)
	if err != nil {
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

func (rm *RepoManager) TrackRepo(repoPath string) (*repo.Repo, error) {
	r, err := rm.openRepo(repoPath)
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
	return r, nil
}

func (rm *RepoManager) openRepo(repoPath string) (*repo.Repo, error) {
	r, err := repo.Open(repoPath)
	if err != nil {
		return nil, err
	}

	repoID, err := r.RepoID()
	if err != nil {
		return nil, err
	}

	if _, exists := rm.repos[repoID]; exists {
		log.Warnf("[repo manager] already opened repo with ID '%v'", repoID)
	}

	rm.repos[repoID] = r
	rm.reposByPath[repoPath] = r

	return r, nil
}

func (rm *RepoManager) UntrackRepo(repoPath string) error {
	r, err := repo.Open(repoPath)
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

func (rm *RepoManager) removeLocalRepoFromConfig(repoPath string) error {
	return rm.config.Update(func() error {
		rm.config.Node.LocalRepos = util.StringSetRemove(rm.config.Node.LocalRepos, repoPath)
		return nil
	})
}

func (rm *RepoManager) Repo(repoID string) *repo.Repo {
	repo, ok := rm.repos[repoID]
	if !ok {
		return nil
	}
	return repo
}

func (rm *RepoManager) RepoAtPath(repoPath string) *repo.Repo {
	repo, ok := rm.reposByPath[repoPath]
	if !ok {
		return nil
	}
	return repo
}

func (rm *RepoManager) RepoAtPathOrID(path string, repoID string) (*repo.Repo, error) {
	if len(path) > 0 {
		r := rm.RepoAtPath(path)
		if r == nil {
			return nil, errors.Errorf("repo at path '%v' not found", path)
		} else {
			return r, nil
		}
	}
	if len(repoID) > 0 {
		r := rm.Repo(repoID)
		if r == nil {
			return nil, errors.Errorf("repo '%v' not found", repoID)
		} else {
			return r, nil
		}
	}

	return nil, errors.Errorf("must provide either 'path' or 'repoID'")
}

func (rm *RepoManager) ForEachRepo(fn func(*repo.Repo) error) error {
	for _, entry := range rm.reposByPath {
		err := fn(entry)
		if err != nil {
			return err
		}
	}
	return nil
}
