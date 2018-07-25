package main

import (
	"fmt"
	// "strings"
	// gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
)

func getRefs() ([]string, error) {
	refs, err := client.GetRefs(repoID)
	if err != nil {
		return nil, err
	}
	refsList := make([]string, 0)
	for name, target := range refs {
		refsList = append(refsList, fmt.Sprintf("%s %s", target, name))
	}
	refsList = append(refsList, fmt.Sprintf("@refs/heads/master HEAD"))
	return refsList, nil
}
