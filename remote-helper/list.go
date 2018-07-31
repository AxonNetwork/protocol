package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	gitconfig "gopkg.in/src-d/go-git.v4/plumbing/format/config"
)

func getRefs() ([]string, error) {
	refs, err := client.GetAllRefs(repoID)
	if err != nil {
		return nil, err
	}
	refsList := make([]string, 0)
	for name, target := range refs {
		refsList = append(refsList, fmt.Sprintf("%s %s", target, name))
	}
	return refsList, nil
}
