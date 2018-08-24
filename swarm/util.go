package swarm

import (
	"github.com/pkg/errors"

	"gx/ipfs/QmPnFwZ2JXKnXgMw8CdBPxn7FWh6LLdjUjxV1fKHuJnkr8/go-multihash"
	"gx/ipfs/QmYVNvtQkeZ6AKSwDrjQTs432QtL6umrrK41EBq3cu7iSP/go-cid"
)

type blankValidator struct{}

func (blankValidator) Validate(_ string, _ []byte) error        { return nil }
func (blankValidator) Select(_ string, _ [][]byte) (int, error) { return 0, nil }

func cidForString(s string) (*cid.Cid, error) {
	pref := cid.NewPrefixV1(cid.Raw, multihash.SHA2_256)
	c, err := pref.Sum([]byte(s))
	if err != nil {
		return nil, errors.Wrap(err, "could not create cid")
	}
	return c, nil
}

func cidForObject(repoID string, objectID []byte) (*cid.Cid, error) {
	pref := cid.NewPrefixV1(cid.Raw, multihash.SHA2_256)
	c, err := pref.Sum(append([]byte(repoID+":"), objectID...))
	if err != nil {
		return nil, errors.Wrap(err, "could not create cid")
	}
	return c, nil
}

func retry(fn func() (bool, error)) error {
	retry, err := fn()
	for retry {
		retry, err = fn()
	}
	return err
}
