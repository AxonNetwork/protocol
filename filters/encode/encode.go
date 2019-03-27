package encode

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/aclements/go-rabin/rabin"
)

const (
	KB                     = 1024
	MB                     = 1024 * KB
	WINDOW_SIZE            = KB
	MIN                    = MB
	AVG                    = 2 * MB
	MAX                    = 4 * MB
	CONSCIENCE_DATA_SUBDIR = "data"
)

func EncodeFile(repoRoot, filename string) (io.ReadCloser, error) {
	p := filepath.Join(repoRoot, filename)
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}

	gitDir := filepath.Join(repoRoot, ".git")
	reader := Encode(gitDir, f)
	return reader, nil
}

func Encode(gitDir string, toRead io.ReadCloser) io.ReadCloser {
	copy := &bytes.Buffer{}
	reader := io.TeeReader(toRead, copy)
	table := rabin.NewTable(rabin.Poly64, WINDOW_SIZE)
	chunker := rabin.NewChunker(table, reader, MIN, AVG, MAX)

	rPipe, wPipe := io.Pipe()

	go func() {
		defer wPipe.Close()
		defer toRead.Close()
		for {
			len, err := chunker.Next()
			if err == io.EOF {
				break
			} else if err != nil {
				wPipe.CloseWithError(err)
				return
			}

			bs := &bytes.Buffer{}
			_, err = io.CopyN(bs, copy, int64(len))
			if err != nil {
				wPipe.CloseWithError(err)
				return
			}

			hash := sha256.Sum256(bs.Bytes())
			hexHash := hex.EncodeToString(hash[:])
			dataRoot := filepath.Join(gitDir, CONSCIENCE_DATA_SUBDIR)
			filePath := filepath.Join(dataRoot, hexHash[:])

			err = os.MkdirAll(dataRoot, 0777)
			if err != nil {
				wPipe.CloseWithError(err)
				return
			}

			if !fileExists(filePath) {
				err = ioutil.WriteFile(filePath, bs.Bytes(), 0666)
				if err != nil {
					wPipe.CloseWithError(err)
					return
				}
			}

			wPipe.Write([]byte(hexHash + "\n"))
		}
	}()

	return rPipe
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
