package wire

import (
	"github.com/libgit2/git2go"
)

type GetPackfileRequest struct {
	RepoIDLen    int `struc:"sizeof=RepoID"`
	RepoID       string
	SignatureLen int `struc:"sizeof=Signature"`
	Signature    []byte
	ObjectIDsLen int `struc:"sizeof=ObjectIDs"`
	ObjectIDs    []byte
}

type GetPackfileResponse struct {
	ErrUnauthorized bool
	ObjectIDsLen    int `struc:"sizeof=ObjectIDs"`
	ObjectIDs       []byte
}

type PackfileStreamChunk struct {
	End     bool
	DataLen int `struc:"sizeof=Data"`
	Data    []byte
}

type GetManifestRequest struct {
	RepoIDLen    int `struc:"sizeof=RepoID"`
	RepoID       string
	Commit       git.Oid
	SignatureLen int `struc:"sizeof=Signature"`
	Signature    []byte
}

type GetManifestResponse struct {
	ErrUnauthorized  bool
	ErrMissingCommit bool
	ManifestLen      int
}

type ManifestObject struct {
	HashLen          int `struc:"sizeof=Hash"`
	Hash             []byte
	UncompressedSize int64
}

type ReplicationRequest struct {
	RepoIDLen int `struc:"sizeof=RepoID"`
	RepoID    string
}

type Progress struct {
	Current     uint64
	Total       uint64
	Done        bool
	Error       error `struc:"skip"`
	ErrorMsgLen int64 `struc:"sizeof=ErrorMsg"`
	ErrorMsg    string
}

type LocalRepo struct {
	RepoID string
	Path   string
}

type Ref struct {
	RefName    string
	CommitHash string
}

type BecomeReplicatorRequest struct {
	RepoIDLen int `struc:"sizeof=RepoID"`
	RepoID    string
}

type BecomeReplicatorResponse struct {
	ErrorLen int64 `struc:"sizeof=Error"`
	Error    string
}
