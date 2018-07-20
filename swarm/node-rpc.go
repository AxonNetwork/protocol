package main

import (
	"fmt"
	"net"
	"net/rpc"
	inet "gx/ipfs/QmPjvxTpVH8qJyQDnxnsxF9kv9jezKD1kozz1hs3fCGsNh/go-libp2p-net"
)

type NodeRPC struct {
	node *Node
}

func RegisterRPC(n *Node, listenPort string) error {
	nr := &NodeRPC{n}
	err := rpc.RegisterName("Node", nr)
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%s", listenPort))
	if err != nil {
		return err
	}

	go rpc.Accept(listener)
	return nil
}

// Input/Output formats for RPC communication from push/pull hooks
type PushHookInput struct {
	RemoteName string
	RemoteUrl  string
	Branch     string
	Commit     string
}

type PushHookOutput struct{}

// For git remote helper
func (nr *NodeRPC) PushHook(in *PushHookInput, out *PushHookOutput) error {
	err := nr.node.PushHook(in.RemoteName, in.RemoteUrl, in.Branch, in.Commit)
	return err
}

type ListHelperInput struct {
	Root string
}

type ListHelperOutput struct{
	Stream inet.Stream
}

func (nr *NodeRPC) ListHelper(in *ListHelperInput, out *ListHelperOutput) error {
	stream, err := nr.node.ListHelper(in.Root)
	out.Stream = stream
	return err
}

// type PullHelperInput struct {
// 	variable string
// }

// type PullHelperOutput struct{}

// func (nr *NodeRPC) PullHelper(in *PullHelperInput, out *PullHelperOutput) error {
// 	err := nr.node.PullHelper(in.variable)
// 	return err
// }
