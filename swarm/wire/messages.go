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

type GetPackfileResponseHeader struct {
	ErrUnauthorized bool
	ObjectIDsLen    int `struc:"sizeof=ObjectIDs"`
	ObjectIDs       []byte
}

type GetPackfileResponsePacket struct {
	DataLen int `struc:"sizeof=Data"`
	Data    []byte
	End     bool
}

type GetChunkRequest struct {
	RepoIDLen    int `struc:"sizeof=RepoID"`
	RepoID       string
	SignatureLen int `struc:"sizeof=Signature"`
	Signature    []byte
	ChunkIDLen   int `struc:"sizeof=ChunkID"`
	ChunkID      []byte
}

type GetChunkResponseHeader struct {
	ErrUnauthorized   bool
	ErrObjectNotFound bool
	Length            int64
}

type GetChunkResponsePacket struct {
	DataLen int `struc:"sizeof=Data"`
	Data    []byte
	End     bool
}

type CheckoutType int

const (
	CheckoutTypeSparse  CheckoutType = 0
	CheckoutTypeWorking CheckoutType = 1
	CheckoutTypeFull    CheckoutType = 2
)

type GetManifestRequest struct {
	RepoIDLen    int `struc:"sizeof=RepoID"`
	RepoID       string
	Commit       git.Oid
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
