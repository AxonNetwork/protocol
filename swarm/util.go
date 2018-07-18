package main

import (
	"strconv"

	"gx/ipfs/QmPnFwZ2JXKnXgMw8CdBPxn7FWh6LLdjUjxV1fKHuJnkr8/go-multihash"
	"gx/ipfs/QmYVNvtQkeZ6AKSwDrjQTs432QtL6umrrK41EBq3cu7iSP/go-cid"
)

func cidFromRepoName(repoName string) (*cid.Cid, error) {
	pref := cid.NewPrefixV1(cid.Raw, multihash.SHA2_256)
	return pref.Sum([]byte(repoName))
}

func incrementPort(p string) (string, error) {
	portInt, err := strconv.ParseInt(p, 10, 64)
	portInt = portInt + 1
	if err != nil {
		return "", nil
	}
	port := strconv.FormatInt(portInt, 10)
	return port, nil
}
