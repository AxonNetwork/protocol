package wire

import (
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
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
	Commit       gitplumbing.Hash
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

type ReplicationProgress struct {
	ErrorLen int64 `struc:"sizeof=Error"`
	Error    string
	Fetched  int64
	ToFetch  int64
	Done     bool
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
