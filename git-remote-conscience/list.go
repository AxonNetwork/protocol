package main

import (
	// "bufio"
	// "bytes"
	// "io/ioutil"
	"encoding/hex"
	"fmt"
	// "strings"

	// "github.com/pkg/errors"
)

func listRefs(repoID string) error {
	objectID, err := hex.DecodeString("089be761cc888ccc50b534d7604f9276f2eafc63")
	if err != nil {
		panic(err)
	}

	//stream, err = ListHelper(repoID, objectID)
	//TODO stream in fetch, as a reader
	if err != nil {
		panic(err)
	} else {
		fmt.Println("All is well")
	}

	return nil
}
