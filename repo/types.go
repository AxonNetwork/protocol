package repo

import (
	// "crypto/sha256"
	"io"

	"github.com/libgit2/git2go"
)

// This is used by various functions to specify a commit.  Either .Hash or .Ref should be non-zero,
// but never both.
type CommitID struct {
	Hash *git.Oid
	Ref  string
}

// const ChunkIDLen = sha256.Size

// type ChunkID [ChunkIDLen]byte

type File struct {
	Filename string
	Hash     git.Oid
	Size     uint64
	Modified uint32
	Status   Status
	// Mode     os.FileMode
	IsChunked bool
}

type Status struct {
	Staged   rune
	Unstaged rune
}

type ObjectReader interface {
	io.ReadCloser
	Len() uint64
	Type() git.ObjectType
}

type objectReader struct {
	io.Reader
	io.Closer
	objectLen  uint64
	objectType git.ObjectType
}

func (or objectReader) Len() uint64 {
	return or.objectLen
}

func (or objectReader) Type() git.ObjectType {
	return or.objectType
}

type FuncCloser func() error

func (fc FuncCloser) Close() error {
	return fc()
}
