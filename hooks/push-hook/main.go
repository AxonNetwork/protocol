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
	cwd, err := os.Getwd()
	check(err)

	client, err := rpc.Dial("tcp", fmt.Sprintf("127.0.0.1:1338"))
	check(err)

	in := AddRepoInput{
		cwd,
	}
	out := AddRepoOutput{}

	err = client.Call("Node.AddRepo", in, &out)
	check(err)
	fmt.Println("Added Repo")

	// remoteName := os.Args[1]
	// remoteUrl := os.Args[2]
	// reader := bufio.NewReader(os.Stdin)
	// branch, err := reader.ReadString(' ')
	// commit, err := reader.ReadString(' ')
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

type Node interface {
	AddRepo(AddRepoInput, *AddRepoOutput) error
}

type AddRepoInput struct {
	RepoPath string
}

type AddRepoOutput struct{}
