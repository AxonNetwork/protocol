package util

import (
	cid "github.com/ipfs/go-cid"
	multihash "github.com/multiformats/go-multihash"

	"github.com/pkg/errors"
)

func CidForString(s string) (cid.Cid, error) {
	pref := cid.NewPrefixV1(cid.Raw, multihash.SHA2_256)
	c, err := pref.Sum([]byte(s))
	if err != nil {
		return cid.Cid{}, errors.Wrap(err, "could not create cid")
	}
	return c, nil
}

func CidForObject(repoID string, objectID []byte) (cid.Cid, error) {
	pref := cid.NewPrefixV1(cid.Raw, multihash.SHA2_256)
	c, err := pref.Sum(append([]byte(repoID+":"), objectID...))
	if err != nil {
		return cid.Cid{}, errors.Wrap(err, "could not create cid")
	}
	return c, nil
}
