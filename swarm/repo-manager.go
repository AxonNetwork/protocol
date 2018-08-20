package swarm

import (
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"../config"
	"../repo"
	"../util"
)

type RepoManager struct {
	repos  map[string]*repo.Repo
	config *config.Config
}

func NewRepoManager(config *config.Config) *RepoManager {
	rm := &RepoManager{
		repos:  make(map[string]*repo.Repo),
		config: config,
	}

	for _, path := range rm.config.Node.LocalRepos {
		log.Infof("[repo manager] tracking local repo: %v", path)
		_, err := rm.openRepo(path)
		if err != nil {
			log.Errorf("[repo manager] %v", err)
			continue
		}
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
	if err != nil {
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

func (rm *RepoManager) ForEachRepo(fn func(*repo.Repo) error) error {
	for _, entry := range rm.repos {
		err := fn(entry)
		if err != nil {
			return err
		}
	}
	return nil
}

func (rm *RepoManager) GetReposInfo() (map[string]repo.RepoInfo, error) {
	repos := make(map[string]repo.RepoInfo)
	err := rm.ForEachRepo(func(r *repo.Repo) error {
		info, err := r.GetInfo()
		if err != nil {
			return err
		}
		repos[info.RepoID] = info
		return nil
	})
	return repos, err
}
