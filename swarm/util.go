package main

import (
	// "encoding/hex"
	"strconv"

	"gx/ipfs/QmPnFwZ2JXKnXgMw8CdBPxn7FWh6LLdjUjxV1fKHuJnkr8/go-multihash"
	"gx/ipfs/QmYVNvtQkeZ6AKSwDrjQTs432QtL6umrrK41EBq3cu7iSP/go-cid"
)

func cidFromString(s string) (*cid.Cid, error) {
	pref := cid.NewPrefixV1(cid.Raw, multihash.SHA2_256)
	return pref.Sum([]byte(s))
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

// type ChunkID struct {
//  RepoID string
//  Hash   ChunkHash
// }

// type Hash interface {
//  String() string
//  Bytes() []byte
//     Len() int
//     MatchesContent([]byte) bool
// }

// type GitHash [20]byte

// func (h GitHash) String() string {
//     return hex.EncodeToString(h[:])
// }

// func (h GitHash) Bytes() []byte {
//     return h[:]
// }

// func (h GitHash) Len() int {
//     return 20
// }

// func (h GitHash) MatchesContent(bs []byte) bool {

// }
