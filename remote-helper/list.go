package main

import (
	"fmt"
)

func getRefs() ([]string, error) {
	refs, err := client.GetAllRefs(repoID)
	if err != nil {
		return nil, err
	}
	refsList := make([]string, 0)
	for _, ref := range refs {
		refsList = append(refsList, fmt.Sprintf("%s %s", ref.Commit, ref.Name))
	}
	refsList = append(refsList, "@refs/heads/master HEAD")
	return refsList, nil
}
