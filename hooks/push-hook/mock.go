// THIS FILE IS ONLY USED FOR TESTING PURPOSES

package main

import (
	"fmt"
	"net/rpc"
	"os"
)

const (
	PORT = "1338"
)

func main() {
	remoteName := os.Args[1]
	remoteUrl := os.Args[2]
	branch := os.Args[3]
	commit := os.Args[4]

	err := GitPush(remoteName, remoteUrl, branch, commit)

	if err != nil {
		panic(err)
	} else {
		fmt.Println("All is well")
	}
}

func GitPush(remoteName string, remoteUrl string, branch string, commit string) error {
	client, err := rpc.Dial("tcp", fmt.Sprintf("127.0.0.1:%s", PORT))
	if err != nil {
		panic(err)
	}

	in := NodeInput{
		remoteName,
		remoteUrl,
		branch,
		commit,
	}
	out := NodeOutput{}
	err = client.Call("Node.GitPush", in, &out)

	return err
}

type Node interface {
	GitPush(*NodeInput, *NodeOutput) error
}

type NodeInput struct {
	RemoteName string
	RemoteUrl  string
	Branch     string
	Commit     string
}

type NodeOutput struct{}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
