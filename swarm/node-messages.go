package swarm

import (
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
)

type MessageType uint64

const (
	MessageType_GetObject MessageType = 0x1
	MessageType_AddRepo   MessageType = 0x2
	MessageType_GetRefs   MessageType = 0x3
	MessageType_AddRef    MessageType = 0x4
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
	Success bool
}

type GetRefsRequest struct {
	RepoIDLen int `struc:"sizeof=RepoID"`
	RepoID    string
}

type GetRefsResponse struct {
	RefsLen int `struc:"sizeof=Refs"`
	Refs    []byte
}

type AddRefRequest struct {
	RepoIDLen int `struc:"sizeof=RepoID"`
	RepoID    string
	NameLen   int `struc:"sizeof=Name"`
	Name      string
	TargetLen int `struc:"sizeof=Target"`
	Target    string
}

type AddRefResponse struct {
	RefsLen int `struc:"sizeof=Refs"`
	Refs    []byte
}
