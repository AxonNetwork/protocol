package util

import (
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
)

func GitHashFromBytes(bs []byte) (hash gitplumbing.Hash) {
	copy(hash[:], bs)
	return
}
