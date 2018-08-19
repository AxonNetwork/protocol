package swarm

import (
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
)

type MessageType uint64

const (
	MessageType_Invalid MessageType = iota
	MessageType_SetUsername
	MessageType_GetObject
	MessageType_CreateRepo
	MessageType_AddRepo
	MessageType_SetReplicationPolicy
	MessageType_AnnounceRepoContent
	MessageType_GetRefs
	MessageType_UpdateRef
	MessageType_Pull
)

type SetUsernameRequest struct {
	UsernameLen int `struc:"sizeof=Username"`
	Username    string
}

type SetUsernameResponse struct {
	ErrorLen int `struc:"sizeof=Error"`
	Error    string
}

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
	ObjectLen    int64
}

type CreateRepoRequest struct {
	RepoIDLen int `struc:"sizeof=RepoID"`
	RepoID    string
}

type CreateRepoResponse struct {
	OK bool
}

type AddRepoRequest struct {
	RepoPathLen int `struc:"sizeof=RepoPath"`
	RepoPath    string
}

type AddRepoResponse struct {
	OK bool
}

type SetReplicationPolicyRequest struct {
	RepoIDLen       int `struc:"sizeof=RepoID"`
	RepoID          string
	ShouldReplicate bool
}

type SetReplicationPolicyResponse struct {
	ErrorLen int `struc:"sizeof=Error"`
	Error    string
}

type AnnounceRepoContentRequest struct {
	RepoIDLen int `struc:"sizeof=RepoID"`
	RepoID    string
}

type AnnounceRepoContentResponse struct {
	OK bool
}

type GetRefsRequest struct {
	RepoIDLen int64 `struc:"sizeof=RepoID"`
	RepoID    string
	Page      int64
}

type GetRefsResponse struct {
	RefsLen int `struc:"sizeof=Refs"`
	Refs    []Ref
	NumRefs int64
}

type UpdateRefRequest struct {
	RepoIDLen  int64 `struc:"sizeof=RepoID"`
	RepoID     string
	RefNameLen int64 `struc:"sizeof=RefName"`
	RefName    string
	CommitLen  int64 `struc:"sizeof=Commit"`
	Commit     string
}

type UpdateRefResponse struct {
	OK bool
}

type Ref struct {
	NameLen   int64 `struc:"sizeof=Name"`
	Name      string
	CommitLen int64 `struc:"sizeof=Commit"`
	Commit    string
}

type PullRequest struct {
	// UsernameLen   int `struc:"sizeof=Username"`
	// Username      string
	RepoIDLen int `struc:"sizeof=RepoID"`
	RepoID    string
}

type PullResponse struct {
	ErrorLen int64 `struc:"sizeof=Error"`
	Error    string
}
