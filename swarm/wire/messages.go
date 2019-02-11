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

type GetPackfileResponsePacket struct {
	End    bool
	Length int
}

type PackfileStreamChunk struct {
	End     bool
	DataLen int `struc:"sizeof=Data"`
	Data    []byte
}

type HandshakeType int

const (
	None        HandshakeType = 0
	ChunkStream HandshakeType = 1
)

type HandshakeRequest struct {
	RepoIDLen     int `struc:"sizeof=RepoID"`
	RepoID        string
	SignatureLen  int `struc:"sizeof=Signature"`
	Signature     []byte
	HandShakeType HandshakeType
}

type HandshakeResponse struct {
	ErrUnauthorized bool
}

type GetChunkRequest struct {
	ChunkIDLen int `struc:"sizeof=ChunkID"`
	ChunkID    []byte
}

type GetChunkResponse struct {
	ErrObjectNotFound bool
	Length            int64
}

type CheckoutType int

const (
	Sparse  CheckoutType = 0
	Working CheckoutType = 1
	Full    CheckoutType = 2
)

type GetManifestRequest struct {
	RepoIDLen    int `struc:"sizeof=RepoID"`
	RepoID       string
	Commit       gitplumbing.Hash
	SignatureLen int `struc:"sizeof=Signature"`
	Signature    []byte
	CheckoutType int
}

type GetManifestResponse struct {
	ErrUnauthorized  bool
	ErrMissingCommit bool
	SendingManifest  bool
}

type ManifestObject struct {
	End              bool
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
