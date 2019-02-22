package gittransport

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/libgit2/git2go"
	"github.com/pkg/errors"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/repo"
	"github.com/Conscience/protocol/swarm/nodep2p"
	"github.com/Conscience/protocol/swarm/wire"
)

type ConscienceTransport struct {
	repoID string
	remote *git.Remote
	node   INode
	repo   *repo.Repo
	wants  []git.RemoteHead
}

type INode interface {
	FetchFromCommit(ctx context.Context, repoID string, repoPath string, commit git.Oid) (<-chan nodep2p.MaybeFetchFromCommitPacket, int64)
	ForEachRemoteRef(ctx context.Context, repoID string, fn func(wire.Ref) (bool, error)) error
}

func Register(node INode) error {
	return git.RegisterTransport("conscience", func(remote *git.Remote) (git.Transport, error) {
		return &ConscienceTransport{remote: remote, node: node}, nil
	})
}

func (t *ConscienceTransport) SetCustomHeaders(headers []string) error {
	return nil
}

func (t *ConscienceTransport) Connect(url string) error {
	log.Warnln("TRANSPORT Connect", t.repoID)
	t.repoID = strings.Replace(url, "conscience://", "", -1)
	return nil
}

func (t *ConscienceTransport) Ls() ([]git.RemoteHead, error) {
	log.Warnln("TRANSPORT Ls", t.repoID)
	// Enumerate the refs from the smart contract
	var headCommitHash string
	var refsList []git.RemoteHead

	err := t.node.ForEachRemoteRef(context.TODO(), t.repoID, func(ref wire.Ref) (bool, error) {
		oid, err := git.NewOid(ref.CommitHash)
		if err != nil {
			return false, err
		}
		refsList = append(refsList, git.RemoteHead{Id: oid, Name: ref.RefName})

		if ref.RefName == "refs/heads/master" {
			headCommitHash = ref.CommitHash
		}

		return true, nil
	})
	if err != nil {
		return nil, err
	} else if len(refsList) == 0 {
		return []git.RemoteHead{}, nil
	} else if headCommitHash == "" {
		return nil, errors.Errorf("ConscienceTransport.Ls: repo %v has no refs/heads/master", t.repoID)
	}

	// Before emitting the concrete refs, we have to emit the symbolic ref for HEAD
	headOid, err := git.NewOid(headCommitHash)
	if err != nil {
		return nil, err
	}
	refsList = append([]git.RemoteHead{{Id: headOid, Name: "HEAD", SymrefTarget: "refs/heads/master"}}, refsList...)

	return refsList, nil
}

func (t *ConscienceTransport) NegotiateFetch(r *git.Repository, wants []git.RemoteHead) error {
	log.Warnln("TRANSPORT NegotiateFetch", t.repoID)
	t.repo = &repo.Repo{Repository: r}
	t.wants = wants
	return nil
}

func (t *ConscienceTransport) DownloadPack(r *git.Repository, progress *git.TransferProgress, progressCb git.TransferProgressCallback) error {
	log.Warnln("TRANSPORT DownloadPack", t.repoID)
	err := t.fetchFromCommit(t.repoID, t.repo, t.wants[0].Id.String(), progressCb)
	if err != nil {
		return err
	}
	return nil
}

func (t *ConscienceTransport) IsConnected() (bool, error) {
	log.Warnln("TRANSPORT IsConnected", t.repoID)
	return true, nil
}

func (t *ConscienceTransport) Cancel() {
	log.Warnln("TRANSPORT Cancel", t.repoID)
}

func (t *ConscienceTransport) Close() error {
	log.Warnln("TRANSPORT Close", t.repoID)
	return nil
}

func (t *ConscienceTransport) Free() {
	log.Warnln("TRANSPORT Free", t.repoID)
}

func (t *ConscienceTransport) fetchFromCommit(repoID string, r *repo.Repo, commitHashStr string, progressCb git.TransferProgressCallback) error {
	commitHash, err := git.NewOid(commitHashStr)
	if err != nil {
		return nil
	} else if r.HasObject(commitHash[:]) {
		return nil
	}

	// @@TODO: give context a timeout and make it configurable
	ch, uncompressedSize := t.node.FetchFromCommit(context.TODO(), repoID, r.Path(), *commitHash)
	if err != nil {
		return err
	}

	// progressWriter := newProgressWriter()
	// fmt.Fprintf(os.Stderr, "\n")

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
				pw, err := r.PackfileWriter()
				if err != nil {
					return err
				}

				packfiles[packfileID] = &PackfileDownload{
					PackfileWriter:   pw,
					uncompressedSize: pkt.PackfileHeader.UncompressedSize,
					written:          0,
				}

				// progressWriter.addDownload(packfileID)
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

		default:
			log.Errorln("bad packet")
		}

		progressCb(git.TransferProgress{TotalObjects: uint(uncompressedSize), ReceivedObjects: uint(written), ReceivedBytes: uint(written)})

		// packfile := packfiles[packfileID]
		// progressWriter.update(packfileID, packfile.written, packfile.uncompressedSize, written, uncompressedSize)
	}
	fmt.Fprint(os.Stderr, "\n")

	return nil
}

// type progressWriter struct {
// 	singleLineWriter *util.SingleLineWriter
// 	multiLineWriter  *util.MultiLineWriter
// 	lines            map[string]int
// }

// func newProgressWriter() *progressWriter {
// 	return &progressWriter{
// 		singleLineWriter: util.NewSingleLineWriter(os.Stderr),
// 		multiLineWriter:  util.NewMultiLineWriter(os.Stderr),
// 		lines:            map[string]int{},
// 	}
// }

// func humanize(x int64) string {
// 	return util.HumanizeBytes(float64(x))
// }

// func (pw *progressWriter) update(packfileID string, packfileWritten, packfileTotal int64, written, total int64) {
// 	// if env.MachineOutputEnabled {
// 	//  pw.singleLineWriter.Printf("Progress: %d/%d ", written, total)
// 	// } else {
// 	pw.multiLineWriter.Printf(0, "Total:      %v %v/%v = %.02f%%", getProgressBar(written, total), humanize(written), humanize(total), 100*(float64(written)/float64(total)))
// 	if packfileWritten == packfileTotal {
// 		pw.multiLineWriter.Printf(pw.lines[packfileID], "pack %v (%v) Done.", packfileID[:6], humanize(packfileTotal))
// 	} else {
// 		pw.multiLineWriter.Printf(pw.lines[packfileID], "pack %v %v %v/%v = %.02f%%", packfileID[:6], getProgressBar(packfileWritten, packfileTotal), humanize(packfileWritten), humanize(packfileTotal), 100*(float64(packfileWritten)/float64(packfileTotal)))
// 	}
// 	// }
// }

// func (pw *progressWriter) addDownload(packfileID string) {
// 	pw.lines[packfileID] = len(pw.lines) + 2
// }

// func getProgressBar(done, total int64) string {
// 	const barWidth = 39

// 	percent := float64(done) / float64(total)
// 	numDashes := int(math.Round(barWidth * percent))
// 	numSpaces := int(math.Round(barWidth * (1 - percent)))

// 	if numDashes+numSpaces > barWidth {
// 		numSpaces--
// 	}

// 	dashes := strings.Repeat("=", numDashes)
// 	spaces := strings.Repeat(" ", numSpaces)

// 	return "[" + dashes + ">" + spaces + "]"
// }
