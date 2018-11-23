package wire

import (
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
)

type MessageType uint64

const (
	MessageType_Invalid MessageType = iota
	MessageType_SetUsername
	MessageType_GetObject
	MessageType_GetManifest
	MessageType_RegisterRepoID
	MessageType_TrackLocalRepo
	MessageType_GetLocalRepos
	MessageType_SetReplicationPolicy
	MessageType_AnnounceRepoContent
	MessageType_GetRefs
	MessageType_UpdateRef
	MessageType_Replicate
	MessageType_BecomeReplicator
)

type GetPackfileRequest struct {
	ObjectIDsLen int `struc:"sizeof=ObjectIDs"`
	ObjectIDs    []byte
}

type GetPackfileResponse struct {
	ObjectIDsLen int `struc:"sizeof=ObjectIDs"`
	ObjectIDs    []byte
}

type PackfileStreamChunk struct {
	End     bool
	DataLen int `struc:"sizeof=Data"`
	Data    []byte
}

type GetObjectRequest struct {
	ObjectIDLen int `struc:"sizeof=ObjectID"`
	ObjectID    []byte
}

type GetObjectRequestSigned struct {
	RepoIDLen    int `struc:"sizeof=RepoID"`
	RepoID       string
	ObjectIDLen  int `struc:"sizeof=ObjectID"`
	ObjectID     []byte
	SignatureLen int `struc:"sizeof=Signature"`
	Signature    []byte
}

type GetObjectResponse struct {
	Unauthorized bool
	HasObject    bool
	ObjectIDLen  int `struc:"sizeof=ObjectID"`
	ObjectID     []byte
	ObjectType   gitplumbing.ObjectType
	ObjectLen    uint64
}

type GetManifestRequest struct {
	RepoIDLen    int `struc:"sizeof=RepoID"`
	RepoID       string
	CommitLen    int `struc:"sizeof=Commit"`
	Commit       string
	SignatureLen int `struc:"sizeof=Signature"`
	Signature    []byte
}

type GetManifestResponse struct {
	Authorized  bool
	HasCommit   bool
	ManifestLen int
}

type ManifestObject struct {
	HashLen int `struc:"sizeof=Hash"`
	Hash    []byte
	Size    int64
}

type HandshakeRequest struct {
	RepoIDLen    int `struc:"sizeof=RepoID"`
	RepoID       string
	SignatureLen int `struc:"sizeof=Signature"`
	Signature    []byte
}

type HandshakeResponse struct {
	Authorized bool
	CommitLen  int `struc:"sizeof=Commit"`
	Commit     string
}

type ReplicationRequest struct {
	RepoIDLen int `struc:"sizeof=RepoID"`
	RepoID    string
}

type ReplicationResponse struct {
	ErrorLen int64 `struc:"sizeof=Error"`
	Error    string
}

type LocalRepo struct {
	RepoID string
	Path   string
}

type Ref struct {
	RefName    string
	CommitHash string
}

type ObjectHeader struct {
	Hash gitplumbing.Hash
	Type gitplumbing.ObjectType
	Len  uint64
}

type ObjectMetadata struct {
	Type gitplumbing.ObjectType
	Len  uint64
}

type BecomeReplicatorRequest struct {
	RepoIDLen int `struc:"sizeof=RepoID"`
	RepoID    string
}

type BecomeReplicatorResponse struct {
	ErrorLen int64 `struc:"sizeof=Error"`
	Error    string
}
