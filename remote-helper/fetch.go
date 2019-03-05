package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/libgit2/git2go"
	"github.com/pkg/errors"

	"github.com/Conscience/protocol/config/env"
	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/repo"
	"github.com/Conscience/protocol/swarm/wire"
	"github.com/Conscience/protocol/util"
)

func fetchFromCommit_packfile(commitHashStr string) error {
	commitHashSlice, err := hex.DecodeString(commitHashStr)
	if err != nil {
		return err
	} else if Repo.HasObject(commitHashSlice) {
		return nil
	}

	var commitHash git.Oid
	copy(commitHash[:], commitHashSlice)

	// @@TODO: give context a timeout and make it configurable
	ch, uncompressedSize, totalChunks, err := client.FetchFromCommit(context.TODO(), repoID, Repo.Path(), commitHash, wire.CheckoutTypeWorking)
	if err != nil {
		return err
	}

	progressWriter := newProgressWriter()
	fmt.Fprintf(os.Stderr, "\n")

	type PackfileDownload struct {
		repo.PackfileWriter
		uncompressedSize int64
		written          int64
	}

	var (
		packfiles     = make(map[string]*PackfileDownload)
		chunks        = make(map[string]*os.File)
		written       int64
		chunksWritten int64
	)

	for pkt := range ch {
		var packfileID string

		switch {
		case pkt.Error != nil:
			return pkt.Error

		case pkt.PackfileHeader != nil:
			packfileID = hex.EncodeToString(pkt.PackfileHeader.PackfileID)

			if _, exists := packfiles[packfileID]; !exists {
				pw, err := Repo.PackfileWriter()
				if err != nil {
					return err
				}

				packfiles[packfileID] = &PackfileDownload{
					PackfileWriter:   pw,
					uncompressedSize: pkt.PackfileHeader.UncompressedSize,
					written:          0,
				}

				progressWriter.addDownload(packfileID)
			}

		case pkt.PackfileData != nil:
			packfileID = hex.EncodeToString(pkt.PackfileData.PackfileID)

			if pkt.PackfileData.End {
				err := packfiles[packfileID].Commit()
				if err != nil {
					packfiles[packfileID].Free()
					return errors.WithStack(err)
				}
				packfiles[packfileID].Free()

				written -= packfiles[packfileID].written          // subtract the compressed byte count from written
				written += packfiles[packfileID].uncompressedSize // add the uncompressed byte count

				packfiles[packfileID].written = packfiles[packfileID].uncompressedSize // we can assume we have the full packfile now, so update `written` to reflect its uncompressed size
				packfiles[packfileID].PackfileWriter = nil                             // don't need the PackfileWriter any longer, let it dealloc

			} else {
				n, err := packfiles[packfileID].Write(pkt.PackfileData.Data)
				if err != nil {
					return errors.WithStack(err)
				} else if n != len(pkt.PackfileData.Data) {
					return errors.New("remote helper: did not fully write packet")
				}

				packfiles[packfileID].written += int64(n)
				written += int64(n)
			}

		case pkt.Chunk != nil:
			objectID := hex.EncodeToString(pkt.Chunk.ObjectID)

			if pkt.Chunk.End {
				err = chunks[objectID].Close()
				if err != nil {
					return errors.WithStack(err)
				}

				chunksWritten++
				chunks[objectID] = nil

			} else {
				f := chunks[objectID]
				if f == nil {
					dataDir := filepath.Join(GIT_DIR, repo.CONSCIENCE_DATA_SUBDIR)
					err := os.MkdirAll(dataDir, 0777)
					if err != nil {
						return errors.WithStack(err)
					}

					f, err = os.Create(filepath.Join(dataDir, objectID))
					if err != nil {
						return errors.WithStack(err)
					}
					chunks[objectID] = f
				}

				n, err := f.Write(pkt.Chunk.Data)
				if err != nil {
					f.Close()
					return errors.WithStack(err)
				} else if n != len(pkt.Chunk.Data) {
					f.Close()
					return errors.New("remote helper: did not fully write chunk")
				}

				written += int64(n)
			}

		default:
			log.Errorln("bad packet")
		}

		progressWriter.updateTotal(written, uncompressedSize)
		if len(packfileID) > 0 {
			packfile := packfiles[packfileID]
			progressWriter.updatePackfile(packfileID, packfile.written, packfile.uncompressedSize)
		} else {
			progressWriter.updateChunks(chunksWritten, totalChunks)
		}

	}
	fmt.Fprint(os.Stderr, "\n")

	return nil
}

type progressWriter struct {
	singleLineWriter *util.SingleLineWriter
	multiLineWriter  *util.MultiLineWriter
	lines            map[string]int
}

func newProgressWriter() *progressWriter {
	return &progressWriter{
		singleLineWriter: util.NewSingleLineWriter(os.Stderr),
		multiLineWriter:  util.NewMultiLineWriter(os.Stderr),
		lines:            map[string]int{},
	}
}

func humanize(x int64) string {
	return util.HumanizeBytes(float64(x))
}

func (pw *progressWriter) updatePackfile(packfileID string, packfileWritten, packfileTotal int64) {
	if !env.MachineOutputEnabled {
		if packfileWritten == packfileTotal {
			pw.multiLineWriter.Printf(pw.lines[packfileID], "pack %v (%v) Done.", packfileID[:6], humanize(packfileTotal))
		} else {
			pw.multiLineWriter.Printf(pw.lines[packfileID], "pack %v %v %v/%v = %.02f%%", packfileID[:6], getProgressBar(packfileWritten, packfileTotal), humanize(packfileWritten), humanize(packfileTotal), 100*(float64(packfileWritten)/float64(packfileTotal)))
		}
	}
}

func (pw *progressWriter) updateChunks(chunksWritten, totalChunks int64) {
	if !env.MachineOutputEnabled {
		pw.multiLineWriter.Printf(2, "Data Chunks:      %v %v/%v = %.02f%%", getProgressBar(chunksWritten, totalChunks), chunksWritten, totalChunks, 100*(float64(chunksWritten)/float64(totalChunks)))
	}
}

func (pw *progressWriter) updateTotal(written, total int64) {
	if env.MachineOutputEnabled {
		pw.singleLineWriter.Printf("Progress: %d/%d ", written, total)
	} else {
		pw.multiLineWriter.Printf(0, "Total:      %v %v/%v = %.02f%%", getProgressBar(written, total), humanize(written), humanize(total), 100*(float64(written)/float64(total)))
	}
}

func (pw *progressWriter) addDownload(packfileID string) {
	pw.lines[packfileID] = len(pw.lines) + 3
}

func getProgressBar(done, total int64) string {
	const barWidth = 39

	percent := float64(done) / float64(total)
	numDashes := int(math.Round(barWidth * percent))
	numSpaces := int(math.Round(barWidth * (1 - percent)))

	if numDashes+numSpaces > barWidth {
		numSpaces--
	}

	dashes := strings.Repeat("=", numDashes)
	spaces := strings.Repeat(" ", numSpaces)

	return "[" + dashes + ">" + spaces + "]"
}
