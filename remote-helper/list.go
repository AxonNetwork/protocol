package main

import (
	"fmt"
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
	return refsList, nil
}
