package main

import (
	"context"
	"fmt"
)

func getRefs() ([]string, error) {
	// @@TODO: give context a timeout and make it configurable
	refs, err := client.GetAllRemoteRefs(context.Background(), repoID)
	if err != nil {
		return nil, err
	} else if len(refs) == 0 {
		return []string{}, nil
	}

	refsList := []string{}
	for _, ref := range refs {
		refsList = append(refsList, fmt.Sprintf("%s %s", ref.CommitHash, ref.RefName))
		// refsList = append(refsList, fmt.Sprintf("56ca5a89f2cf4b398f8ee2e755bbf71e1559b6b2 %s", ref.RefName))
	}
	refsList = append(refsList, "@refs/heads/master HEAD")
	return refsList, nil
}
