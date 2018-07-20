package main

import (
	"fmt"
	"net/rpc"
	// "encoding/hex"
	// "os"
	inet "gx/ipfs/QmPjvxTpVH8qJyQDnxnsxF9kv9jezKD1kozz1hs3fCGsNh/go-libp2p-net"
)

const (
	PORT = "1338"
)

// func main() {
// 	repoID := os.Args[1]
// 	hexHash := os.Args[2]

// 	objectID, err := hex.DecodeString(hexHash)
// 	if err != nil {
// 		panic(err)
// 	}

// 	err = ListHelper(repoID, objectID)

// 	if err != nil {
// 		panic(err)
// 	} else {
// 		fmt.Println("All is well")
// 	}
// }

func ListHelper(repoID string, objectID []byte) (*inet.Stream, error) {
	client, err := rpc.Dial("tcp", fmt.Sprintf("127.0.0.1:%s", PORT))

	fmt.Printf("cient: %v", client)
	if err != nil {
		panic(err)
	}

	in := ListHelperInput{
		repoID,
		objectID,
	}
	out := ListHelperOutput{
		nil,
	}
	err = client.Call("Node.ListHelper", in, &out)

	fmt.Printf("out after call: %v", out.Stream)

	return out.Stream, err
}

type Node interface {
	ListHelper(*ListHelperInput, *ListHelperOutput) error
}

type ListHelperInput struct {
	RepoID string
	ObjectID []byte
}

type ListHelperOutput struct{
	Stream *inet.Stream
}

// func check(e error) {
// 	if e != nil {
// 		panic(e)
// 	}
// }
