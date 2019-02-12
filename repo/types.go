package repo

import (
	"os"

	git "gopkg.in/src-d/go-git.v4"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
)

// This is used by various functions to specify a commit.  Either .Hash or .Ref should be non-zero,
// but never both.
type CommitID struct {
	Hash gitplumbing.Hash
	Ref  string
}

type File struct {
	Filename string
	Hash     gitplumbing.Hash
	Status   git.FileStatus
	Size     uint64
	Mode     os.FileMode
	Modified uint32
}
