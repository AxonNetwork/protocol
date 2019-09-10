package decode

import (
	"bufio"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"

	"github.com/Conscience/protocol/repo"
)

func Decode(r *repo.Repo, ioReader io.Reader, fetchChunks func(chunks [][]byte) error) io.ReadCloser {
	rPipe, wPipe := io.Pipe()
	chunks := make([]string, 0)
	toFetch := make([][]byte, 0)
	reader := bufio.NewReader(ioReader)

	go func() {
		defer wPipe.Close()
		for {
			line, _, err := reader.ReadLine()
			if err == io.EOF {
				break
			} else if err != nil {
				wPipe.CloseWithError(err)
				return
			}

			p := filepath.Join(r.ChunkDir(), string(line))
			chunks = append(chunks, p)
			_, err = os.Stat(p)
			if os.IsNotExist(err) {
				hash, err := hex.DecodeString(string(line))
				if err != nil {
					wPipe.CloseWithError(err)
					return
				}
				toFetch = append(toFetch, hash)
			} else if err != nil {
				wPipe.CloseWithError(err)
				return
			}
		}

		if len(toFetch) > 0 {
			err := fetchChunks(toFetch)
			if err != nil {
				wPipe.CloseWithError(err)
				return
			}
		}

		for _, chunk := range chunks {
			f, err := os.Open(chunk)
			if err != nil {
				wPipe.CloseWithError(err)
				return
			}
			_, err = io.Copy(wPipe, f)
			if err != nil {
				wPipe.CloseWithError(err)
				return
			}
			f.Close()
		}
	}()

	return rPipe
}
