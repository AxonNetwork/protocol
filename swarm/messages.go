package swarm

import (
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
)

type MessageType uint64

const (
	MessageType_Invalid MessageType = iota
	MessageType_GetObject
	MessageType_AddRepo
	MessageType_AnnounceRepoContent
	MessageType_GetRefs
	MessageType_UpdateRef
	MessageType_Pull
)

type GetObjectRequest struct {
	RepoIDLen   int `struc:"sizeof=RepoID"`
	RepoID      string
	ObjectIDLen int `struc:"sizeof=ObjectID"`
	ObjectID    []byte
}

type GetObjectResponse struct {
	HasObject  bool
	ObjectType gitplumbing.ObjectType
	ObjectLen  int64
}

type AddRepoRequest struct {
	RepoPathLen int `struc:"sizeof=RepoPath"`
	RepoPath    string
}

type AddRepoResponse struct {
	OK bool
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
	OK bool
}
