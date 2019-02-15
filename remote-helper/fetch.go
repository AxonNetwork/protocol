package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/libgit2/git2go"
	"github.com/pkg/errors"

	"github.com/Conscience/protocol/config/env"
	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/repo"
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
	ch, uncompressedSize, err := client.FetchFromCommit(context.TODO(), repoID, Repo.Path, commitHash)
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

	packfiles := make(map[string]*PackfileDownload)
	var written int64

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
				_, err = packfiles[packfileID].Commit()
				if err != nil {
					packfiles[packfileID].Free()
					return errors.WithStack(err)
				}
				packfiles[packfileID].Free()

				written -= packfiles[packfileID].written          // subtract the compressed byte count from written
				written += packfiles[packfileID].uncompressedSize // add the uncompressed byte count

				packfiles[packfileID].written = packfiles[packfileID].uncompressedSize // we can assume we have the full packfile now, so update `written` to reflect its uncompressed size
				packfiles[packfileID].WriteCloser = nil                                // don't need the io.WriteCloser any longer, let it dealloc

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

		default:
			log.Errorln("bad packet")
		}

		packfile := packfiles[packfileID]
		progressWriter.update(packfileID, packfile.written, packfile.uncompressedSize, written, uncompressedSize)
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

func (pw *progressWriter) update(packfileID string, packfileWritten, packfileTotal int64, written, total int64) {
	if env.MachineOutputEnabled {
		pw.singleLineWriter.Printf("Progress: %d/%d ", written, total)
	} else {
		pw.multiLineWriter.Printf(0, "Total:      %v %v/%v = %.02f%%", getProgressBar(written, total), humanize(written), humanize(total), 100*(float64(written)/float64(total)))
		if packfileWritten == packfileTotal {
			pw.multiLineWriter.Printf(pw.lines[packfileID], "pack %v (%v) Done.", packfileID[:6], humanize(packfileTotal))
		} else {
			pw.multiLineWriter.Printf(pw.lines[packfileID], "pack %v %v %v/%v = %.02f%%", packfileID[:6], getProgressBar(packfileWritten, packfileTotal), humanize(packfileWritten), humanize(packfileTotal), 100*(float64(packfileWritten)/float64(packfileTotal)))
		}
	}
}

func (pw *progressWriter) addDownload(packfileID string) {
	pw.lines[packfileID] = len(pw.lines) + 2
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
