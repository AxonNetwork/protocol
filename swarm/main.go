package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	peer "gx/ipfs/QmdVrMn1LhB4ybb8hMVaMLXnA8XRSewMnK6YqXKXoTcRvN/go-libp2p-peer"
)

func main() {
	listenPort := os.Args[1]

	ctx := context.Background()

	n, err := NewNode(ctx, listenPort)
	if err != nil {
		panic(err)
	}

	inputLoop(ctx, n)
}

func inputLoop(ctx context.Context, n *Node) {
	fmt.Printf("> ")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, " ")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}

		var err error

		switch parts[0] {
		case "addrs":
			logAddrs(n)

		case "repos":
			logRepos(n)

		case "peers":
			logPeers(n)

		case "bootstrap":
			err = n.Bootstrap(ctx)

		case "add-repo":
			if len(parts) < 2 {
				err = fmt.Errorf("not enough args")
				break
			}
			err = n.RepoManager.AddRepo(parts[1])

		case "add-peer":
			if len(parts) < 2 {
				err = fmt.Errorf("not enough args")
				break
			}
			err = n.AddPeer(ctx, parts[1])

		case "get":
			if len(parts) < 2 {
				err = fmt.Errorf("not enough args")
				break
			}
			_, err = n.GetValue(ctx, parts[1])

		case "set":
			if len(parts) < 3 {
				err = fmt.Errorf("not enough args")
				break
			}
			err = n.SetValue(ctx, parts[1], parts[2])

		case "provide":
			if len(parts) < 2 {
				err = fmt.Errorf("not enough args")
				break
			}
			err = n.Provide(ctx, parts[1])

		case "find-providers":
			if len(parts) < 2 {
				err = fmt.Errorf("not enough args")
				break
			}
			_, err = n.FindProviders(ctx, parts[1])

		case "get-chunk":
			if len(parts) < 4 {
				err = fmt.Errorf("not enough args")
				break
			}
			var hasChunk bool
			hasChunk, err = n.GetChunk(ctx, parts[1], parts[2], parts[3])
			log.Printf("has chunk? %v", hasChunk)

		case "get-repo":
			if len(parts) < 3 {
				err = fmt.Errorf("not enough args")
				break
			}
			var hasRepo bool
			hasRepo, err = n.GetRepo(ctx, parts[1], parts[2])
			log.Printf("has repo? %v", hasRepo)

		default:
			err = fmt.Errorf("unknown command")
		}

		if err != nil {
			log.Errorln(err)
		}

		fmt.Printf("> ")
	}
}

func logPeers(n *Node) {
	log.Printf("total connected peers: %v", len(n.Host.Network().Conns()))

	for _, peerID := range n.Host.Peerstore().Peers() {
		log.Printf("  - %v (%v)", peerID.String(), peer.IDB58Encode(peerID))
		for _, addr := range n.Host.Peerstore().Addrs(peerID) {
			log.Printf("      - %v", addr)
		}
	}
}

func logAddrs(n *Node) {
	for _, addr := range n.Host.Addrs() {
		log.Println(addr.String() + "/ipfs/" + n.Host.ID().Pretty())
	}
}

func logRepos(n *Node) {
	log.Printf("Known repos:")
	for _, repoName := range n.RepoManager.RepoNames() {
		log.Printf("  - %v", repoName)
		for _, chunk := range n.RepoManager.ChunksForRepo(repoName) {
			log.Printf("      - %v", chunk)
		}
	}
}
