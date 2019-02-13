package repo

import (
	"io"
	"os"

	git "gopkg.in/src-d/go-git.v4"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
)

type GitHash = gitplumbing.Hash

var ZeroHash = gitplumbing.ZeroHash

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

type ObjectReader interface {
	io.ReadCloser
	Len() uint64
	Type() gitplumbing.ObjectType
}

type objectReader struct {
	io.Reader
	io.Closer
	objectLen  uint64
	objectType gitplumbing.ObjectType
}

func (or objectReader) Len() uint64 {
	return or.objectLen
}

func (or objectReader) Type() gitplumbing.ObjectType {
	return or.objectType
}
