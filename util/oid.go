package util

import (
	"github.com/libgit2/git2go"
)

func OidFromBytes(bs []byte) *git.Oid {
	var oid git.Oid
	copy(oid[:], bs)
	return &oid
}
