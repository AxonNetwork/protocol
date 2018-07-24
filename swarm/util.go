package swarm

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/lunixbochs/struc"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"gx/ipfs/QmPnFwZ2JXKnXgMw8CdBPxn7FWh6LLdjUjxV1fKHuJnkr8/go-multihash"
	"gx/ipfs/QmYVNvtQkeZ6AKSwDrjQTs432QtL6umrrK41EBq3cu7iSP/go-cid"
)

var (
	ErrUnexpectedEOF = errors.New("unexpected EOF")
)

func cidForString(s string) (*cid.Cid, error) {
	pref := cid.NewPrefixV1(cid.Raw, multihash.SHA2_256)
	return pref.Sum([]byte(s))
}

func cidForObject(repoID string, objectID []byte) (*cid.Cid, error) {
	pref := cid.NewPrefixV1(cid.Raw, multihash.SHA2_256)
	return pref.Sum(append([]byte(repoID+":"), objectID...))
}

func readUint64(r io.Reader) (uint64, error) {
	buf := make([]byte, 8)
	_, err := io.ReadFull(r, buf)
	if err != nil {
		return 0, errors.Wrap(ErrUnexpectedEOF, "readUint64")
	}
	return binary.LittleEndian.Uint64(buf), nil
}

func writeUint64(w io.Writer, n uint64) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, n)
	written, err := w.Write(buf)
	if err != nil {
		return err
	} else if written < 8 {
		return errors.Wrap(ErrUnexpectedEOF, "writeUint64")
	}
	return nil
}

func writeStructPacket(w io.Writer, obj interface{}) error {
	buf := &bytes.Buffer{}

	err := struc.Pack(buf, obj)
	if err != nil {
		return err
	}

	buflen := buf.Len()
	err = writeUint64(w, uint64(buflen))
	if err != nil {
		return err
	}

	n, err := io.Copy(w, buf)
	if err != nil {
		return err
	} else if n != int64(buflen) {
		log.Printf("writeStructPacket ERR: could not write entire packet")
		return fmt.Errorf("writeStructPacket: could not write entire packet")
	}
	log.Printf("writeStructPacket: (%v bytes) %+v", buflen, obj)
	return nil
}

func readStructPacket(r io.Reader, obj interface{}) error {
	size, err := readUint64(r)
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
	log.Printf("readStructPacket: (%v bytes) %+v", size, obj)
	return nil
}

// type ChunkID struct {
//  RepoID string
//  Hash   ChunkHash
// }

// type Hash interface {
//  String() string
//  Bytes() []byte
//     Len() int
//     MatchesContent([]byte) bool
// }

// type GitHash [20]byte

// func (h GitHash) String() string {
//     return hex.EncodeToString(h[:])
// }

// func (h GitHash) Bytes() []byte {
//     return h[:]
// }

// func (h GitHash) Len() int {
//     return 20
// }

// func (h GitHash) MatchesContent(bs []byte) bool {

// }
