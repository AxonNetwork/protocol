package swarm

import (
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
)

type MessageType uint64

const (
	MessageType_GetObject MessageType = 0x1
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
