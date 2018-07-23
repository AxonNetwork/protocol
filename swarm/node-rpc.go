package swarm

import (
	"fmt"
	"net"
	"net/rpc"

	inet "gx/ipfs/QmPjvxTpVH8qJyQDnxnsxF9kv9jezKD1kozz1hs3fCGsNh/go-libp2p-net"
)

type NodeRPC struct {
	node *Node
}

func RegisterRPC(n *Node, listenPort int) error {
	nr := &NodeRPC{n}
	err := rpc.RegisterName("Node", nr)
	if err != nil {
		return err
	}

	fmt.Println("rpc port: ", listenPort)

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%v", listenPort))
	if err != nil {
		return err
	}

	go rpc.Accept(listener)
	return nil
}

// ********************
// AddRepo
// ********************

type AddRepoInput struct {
	RepoPath string
}

type AddRepoOutput struct{}

func (nr *NodeRPC) AddRepo(in *AddRepoInput, out *AddRepoOutput) error {
	err := nr.node.RepoManager.AddRepo(in.RepoPath)
	return err
}

// ********************
// GetObject
// ********************

// func (nr *NodeRPC) GetObject(in *noderpc.GetObjectInput, out *noderpc.GetObjectOutput) error {
//  ctx := context.Background()
//  err := nr.node.GetObject(ctx, in.RepoID, in.ObjectID)
//  return err
// }

// ********************
// ListHelper
// ********************

type ListHelperInput struct {
	RepoID   string
	ObjectID []byte
}

type ListHelperOutput struct {
	Stream inet.Stream
}

func (nr *NodeRPC) ListHelper(in *ListHelperInput, out *ListHelperOutput) error {
	stream, err := nr.node.ListHelper(in.RepoID, in.ObjectID)
	out.Stream = stream
	return err
}
