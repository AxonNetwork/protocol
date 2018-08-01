package main

import (
	"strings"

	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
)

func push(refName string, commitHash string) error {
	force := strings.HasPrefix(refName, "+")
	if force {
		refName = refName[1:]
	}

	err := client.AnnounceRepoContent(repoID)
	if err != nil {
		return err
	}

	ref, err := Repo.Reference(gitplumbing.ReferenceName(refName), false)
	if err != nil {
		return err
	}

	err = client.UpdateRef(repoID, ref.Strings()[1], commitHash)
	if err != nil {
		return err
	}

	err = client.RequestPull(repoID)
	return err
}
