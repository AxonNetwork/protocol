package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/pkg/errors"

	"github.com/Conscience/protocol/util"
)

func fetchFromCommit_packfile(commitHash string) error {
	hash, err := hex.DecodeString(commitHash)
	if err != nil {
		return err
	} else if Repo.HasObject(hash) {
		return nil
	}

	ch, uncompressedSize, err := client.FetchFromCommit(context.Background(), repoID, Repo.Path, commitHash)
	if err != nil {
		return err
	}

	progressWriter := util.NewSingleLineWriter(os.Stderr)

	type PackfileDownload struct {
		io.WriteCloser
		uncompressedSize int64
		written          int64
	}

	packfiles := make(map[string]PackfileDownload)
	var writtenBytes int64

	for pkt := range ch {
		switch {
		case pkt.Error != nil:
			return pkt.Error

		case pkt.PackfileHeader != nil:
			packfileID := hex.EncodeToString(pkt.PackfileHeader.PackfileID)

			if _, exists := packfiles[packfileID]; !exists {
				pw, err := Repo.PackfileWriter()
				if err != nil {
					return err
				}

				packfiles[packfileID] = PackfileDownload{
					WriteCloser:      pw,
					uncompressedSize: pkt.PackfileHeader.UncompressedSize,
					written:          0,
				}
			}

		case pkt.PackfileData != nil:
			packfileID := hex.EncodeToString(pkt.PackfileData.ObjHash)

			if pkt.PackfileData.End {
				err = packfiles[packfileID].Close()
				if err != nil {
					return errors.WithStack(err)
				}

				writtenBytes -= packfiles[packfileID].written          // subtract the compressed byte count from writtenBytes
				writtenBytes += packfiles[packfileID].uncompressedSize // add the uncompressed byte count

				delete(packfiles, packfileID)

			} else {
				n, err := packfiles[packfileID].Write(pkt.PackfileData.Data)
				if err != nil {
					return errors.WithStack(err)
				} else if n != len(pkt.PackfileData.Data) {
					return errors.New("remote helper: did not fully write packet")
				}

				x := packfiles[packfileID]
				x.written += int64(n)
				packfiles[packfileID] = x

				writtenBytes += int64(n)
			}
		}

		progressWriter.Printf("Progress: %v/%v = %.02f%%", writtenBytes, uncompressedSize, 100*(float64(writtenBytes)/float64(uncompressedSize)))
	}
	fmt.Fprint(os.Stderr, "\n")

	return nil
}
