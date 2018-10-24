package wire

import (
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
)

type MessageType uint64

const (
	MessageType_Invalid MessageType = iota
	MessageType_SetUsername
	MessageType_GetObject
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

type GetObjectRequest struct {
	RepoIDLen   int `struc:"sizeof=RepoID"`
	RepoID      string
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
	ObjectType   gitplumbing.ObjectType
	ObjectLen    uint64
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
