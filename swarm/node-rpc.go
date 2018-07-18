package main

import (
	"fmt"
	"net"
	"net/rpc"
)

type NodeRPC struct {
	node *Node
}

// Input/Output formats for RPC communication from push/pull hooks
type NodeInput struct {
	RemoteName string
	RemoteUrl  string
	Branch     string
	Commit     string
}

type NodeOutput struct{}

func (nr *NodeRPC) GitPush(in *NodeInput, out *NodeOutput) error {
	err := nr.node.GitPush(in.RemoteName, in.RemoteUrl, in.Branch, in.Commit)
	return err
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
