package util

import (
	multihash "gx/ipfs/QmPnFwZ2JXKnXgMw8CdBPxn7FWh6LLdjUjxV1fKHuJnkr8/go-multihash"
	cid "gx/ipfs/QmYVNvtQkeZ6AKSwDrjQTs432QtL6umrrK41EBq3cu7iSP/go-cid"

	"github.com/pkg/errors"
)

func CidForString(s string) (*cid.Cid, error) {
	pref := cid.NewPrefixV1(cid.Raw, multihash.SHA2_256)
	c, err := pref.Sum([]byte(s))
	if err != nil {
		return nil, errors.Wrap(err, "could not create cid")
	}
	return c, nil
}

func CidForObject(repoID string, objectID []byte) (*cid.Cid, error) {
	pref := cid.NewPrefixV1(cid.Raw, multihash.SHA2_256)
	c, err := pref.Sum(append([]byte(repoID+":"), objectID...))
	if err != nil {
		return nil, errors.Wrap(err, "could not create cid")
	}
	return c, nil
}
