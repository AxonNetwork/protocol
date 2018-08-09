package swarm

import (
	"../repo"
)

type RepoManager struct {
	repos map[string]*repo.Repo
}

func NewRepoManager() *RepoManager {
	return &RepoManager{
		repos: make(map[string]*repo.Repo),
	}
}

func (rm *RepoManager) AddRepo(repoPath string) (*repo.Repo, error) {
	r, err := repo.Open(repoPath)
	if err != nil {
		return nil, err
	}

	repoID, err := r.RepoID()
	if err != nil {
		return nil, err
	}

	rm.repos[repoID] = r
	return r, nil
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
