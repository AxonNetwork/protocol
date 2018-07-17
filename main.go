package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

func main() {
	listenPort := os.Args[1]

	ctx := context.Background()

	n, err := NewNode(ctx, listenPort)
	if err != nil {
		panic(err)
	}

	log.Println("peerID is: ", n.Host.ID().Pretty())

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
		case "bootstrap":
			err = n.Bootstrap(ctx)

		case "peers":
			err = n.LogPeers()

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
			err = n.GetValue(ctx, parts[1])

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
			err = n.FindProviders(ctx, parts[1])

		case "stream":
			if len(parts) < 3 {
				err = fmt.Errorf("not enough args")
				break
			}
			err = n.SendChunkStream(ctx, parts[1], parts[2])

		default:
			err = fmt.Errorf("unknown command")
		}

		if err != nil {
			log.Errorln(err)
		}

		fmt.Printf("> ")
	}
}
