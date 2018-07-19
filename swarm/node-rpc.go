package main

import (
	"fmt"
	"net"
	"net/rpc"
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

func (nr *NodeRPC) PushHook(in *NodeInput, out *NodeOutput) error {
	err := nr.node.PushHook(in.RemoteName, in.RemoteUrl, in.Branch, in.Commit)
	return err
}

type PushHelperInput struct {
	variable string
}

type PushHelperOutput struct{}

func (nr *NodeRPC) PushHelper(in *NodeInput, out *NodeOutput) error {
	err := nr.node.PushHelper(in.variable)
	return err
}

type PullHelperInput struct {
	variable string
}

type PullHelperOutput struct{}

func (nr *NodeRPC) PullHelper(in *NodeInput, out *NodeOutput) error {
	err := nr.node.PullHelper(in.variable)
	return err
}
