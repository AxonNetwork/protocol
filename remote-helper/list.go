package main

import (
	"context"
	"fmt"
)

func getRefs() ([]string, error) {
	// @@TODO: give context a timeout and make it configurable
	refs, err := client.GetAllRefs(context.Background(), repoID)
	if err != nil {
		return nil, err
	}
	refsList := make([]string, 0)
	for _, ref := range refs {
		refsList = append(refsList, fmt.Sprintf("%s %s", ref.CommitHash, ref.RefName))
	}
	refsList = append(refsList, "@refs/heads/master HEAD")
	return refsList, nil
}
