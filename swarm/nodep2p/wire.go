package nodep2p

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"sort"

	"github.com/libgit2/git2go"

	"github.com/lunixbochs/struc"
	"github.com/pkg/errors"
)

var (
	ErrUnexpectedEOF  = errors.New("unexpected EOF")
	ErrUnauthorized   = errors.New("not authorized to pull object")
	ErrObjectNotFound = errors.New("object not found")
	ErrProtocol       = errors.New("protocol error")
)

type GetPackfileRequest struct {
	RepoIDLen    int `struc:"sizeof=RepoID"`
	RepoID       string
	SignatureLen int `struc:"sizeof=Signature"`
	Signature    []byte
	ObjectIDsLen int `struc:"sizeof=ObjectIDs"`
	ObjectIDs    []byte
}

type GetPackfileResponseHeader struct {
	ErrUnauthorized bool
	ObjectIDsLen    int `struc:"sizeof=ObjectIDs"`
	ObjectIDs       []byte
}

type GetPackfileResponsePacket struct {
	DataLen int `struc:"sizeof=Data"`
	Data    []byte
	End     bool
}

type GetChunkRequest struct {
	RepoIDLen    int `struc:"sizeof=RepoID"`
	RepoID       string
	SignatureLen int `struc:"sizeof=Signature"`
	Signature    []byte
	ChunkIDLen   int `struc:"sizeof=ChunkID"`
	ChunkID      []byte
}

type GetChunkResponseHeader struct {
	ErrUnauthorized   bool
	ErrObjectNotFound bool
	Length            int64
}

type GetChunkResponsePacket struct {
	DataLen int `struc:"sizeof=Data"`
	Data    []byte
	End     bool
}

type CheckoutType int

const (
	// A 'sparse' checkout includes no large file chunks.  Chunked files are left un-decoded.
	CheckoutTypeSparse CheckoutType = 0
	// A 'working' checkout includes all normal git objects, but only the large file chunks associated with the current HEAD commit
	CheckoutTypeWorking CheckoutType = 1
	// A 'full' checkout includes all normal git objects and all large file chunks.
	CheckoutTypeFull CheckoutType = 2
)

type GetManifestRequest struct {
	RepoIDLen    int `struc:"sizeof=RepoID"`
	RepoID       string
	Commit       git.Oid
	SignatureLen int `struc:"sizeof=Signature"`
	Signature    []byte
	CheckoutType int
}

type GetManifestResponse struct {
	ErrUnauthorized  bool
	ErrMissingCommit bool
	SendingManifest  bool
}

type ReplicationRequest struct {
	RepoIDLen int `struc:"sizeof=RepoID"`
	RepoID    string
}

type Progress struct {
	Current     uint64
	Total       uint64
	Done        bool
	Error       error `struc:"skip"`
	ErrorMsgLen int64 `struc:"sizeof=ErrorMsg"`
	ErrorMsg    string
}

type LocalRepo struct {
	RepoID string
	Path   string
}

type Manifest struct {
	GitObjects   ManifestObjectSet
	ChunkObjects ManifestObjectSet
}

type ManifestObjectSet []ManifestObject

type ManifestObject struct {
	End              bool
	HashLen          int `struc:"sizeof=Hash"`
	Hash             []byte
	UncompressedSize int64
}

func (s ManifestObjectSet) UncompressedSize() int64 {
	var size int64
	for i := range s {
		size += s[i].UncompressedSize
	}
	return size
}

func (s ManifestObjectSet) ToHashes() [][]byte {
	hashes := make([][]byte, 0)
	for _, obj := range s {
		hashes = append(hashes, obj.Hash)
	}
	return hashes
}

func ReadUint64(r io.Reader) (uint64, error) {
	buf := make([]byte, 8)
	_, err := io.ReadFull(r, buf)
	if err == io.EOF {
		return 0, err
	} else if err != nil {
		return 0, errors.Wrap(err, "ReadUint64")
	}
	return binary.LittleEndian.Uint64(buf), nil
}

func WriteUint64(w io.Writer, n uint64) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, n)
	written, err := w.Write(buf)
	if err != nil {
		return err
	} else if written < 8 {
		return errors.Wrap(err, "WriteUint64")
	}
	return nil
}

func WriteMsg(w io.Writer, obj interface{}) error {
	buf := &bytes.Buffer{}

	err := struc.Pack(buf, obj)
	if err != nil {
		return err
	}

	buflen := buf.Len()
	err = WriteUint64(w, uint64(buflen))
	if err != nil {
		return err
	}

	n, err := io.Copy(w, buf)
	if err != nil {
		return err
	} else if n != int64(buflen) {
		return fmt.Errorf("WriteMsg: could not write entire packet")
	}
	return nil
}

func ReadMsg(r io.Reader, obj interface{}) error {
	size, err := ReadUint64(r)
	if err != nil {
		return err
	}

	buf := &bytes.Buffer{}
	_, err = io.CopyN(buf, r, int64(size))
	if err != nil {
		return err
	}

	err = struc.Unpack(buf, obj)
	if err != nil {
		return err
	}
	return nil
}

func FlattenObjectIDs(objectIDs [][]byte) []byte {
	flattened := []byte{}
	for i := range objectIDs {
		flattened = append(flattened, objectIDs[i]...)
	}
	return flattened
}

func UnflattenObjectIDs(flattened []byte) [][]byte {
	numObjects := len(flattened) / 20
	objectIDs := make([][]byte, numObjects)
	for i := 0; i < numObjects; i++ {
		objectIDs[i] = flattened[i*20 : (i+1)*20]
	}
	return objectIDs
}

type sortByteSlices [][]byte

func (b sortByteSlices) Len() int {
	return len(b)
}

func (b sortByteSlices) Less(i, j int) bool {
	switch bytes.Compare(b[i], b[j]) {
	case -1:
		return true
	case 0, 1:
		return false
	}
	panic("bytes.Compare did not return (-1,0,1)")
}

func (b sortByteSlices) Swap(i, j int) {
	b[j], b[i] = b[i], b[j]
}

func SortByteSlices(src [][]byte) [][]byte {
	sorted := sortByteSlices(src)
	sort.Sort(sorted)
	return sorted
}
