package main

import (
	"fmt"
	"os"
)

func main() {
	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	var err error

	switch cmd {
	case "init":
		if len(os.Args) < 3 {
			fmt.Println("\nUsage: conscience init <repoid>\n")
			break
		}
		err = initRepo(os.Args[2])

	default:
		if len(cmd) > 0 {
			fmt.Println("\nInvalid command")
		}
		fmt.Println("\nAvailable commands:")
		fmt.Println(" - conscience init <repoid>\n")
		fmt.Println("")
	}

	if err != nil {
		panic(err)
	}
}
