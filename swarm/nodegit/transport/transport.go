package transport

/*
#include <git2.h>
#include <git2/sys/transport.h>

*/
import "C"
import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/libgit2/git2go"
	"github.com/pkg/errors"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/repo"
	"github.com/Conscience/protocol/swarm/noderpc"
	"github.com/Conscience/protocol/util"
)

func init() {
	err := git.RegisterTransport("conscience", func(remote *git.Remote) (git.Transport, error) {
		return &ConscienceTransport{remote: remote}, nil
	})
	if err != nil {
		panic(err)
	}
}

func TransportTest(url, path string) error {
	_, err := git.Clone(url, path, &git.CloneOptions{}) /*&git.CloneOptions{
	    RemoteCreateCallback: func(r *git.Repository, name, url string) (*git.Remote, git.ErrorCode) {
	        remote, err := r.Remotes.Create("origin", url)
	        if err != nil {
	            return nil, git.ErrGeneric
	        }

	        return remote, git.ErrOk
	    },
	}*/
	if err != nil {
		return err
	}
	return nil
}

type ConscienceTransport struct {
	repoID string
	remote *git.Remote
	client *noderpc.Client

	wants []git.RemoteHead
}

func (t *ConscienceTransport) SetCustomHeaders(headers []string) error {
	fmt.Println(t.repoID, "SetCustomHeaders", headers)
	return nil
}

func (t *ConscienceTransport) Connect(url string) error {
	fmt.Println(t.repoID, "Connect", url)

	t.repoID = strings.Replace(url, "conscience://", "", -1)

	client, err := noderpc.NewClient("0.0.0.0:1338")
	if err != nil {
		return err
	}

	t.client = client

	return nil
}

func (t *ConscienceTransport) Ls() ([]git.RemoteHead, error) {
	fmt.Println(t.repoID, "Ls")

	refs, err := t.client.GetAllRemoteRefs(context.Background(), t.repoID)
	if err != nil {
		return nil, err
	} else if len(refs) == 0 {
		return []git.RemoteHead{}, nil
	}

	refsList := []git.RemoteHead{}
	for _, ref := range refs {
		oid, err := git.NewOid(ref.CommitHash)
		if err != nil {
			return nil, err
		}
		refsList = append(refsList, git.RemoteHead{Id: oid, Name: ref.RefName})
		fmt.Println("ref:", refsList[len(refsList)-1].Id, refsList[len(refsList)-1].Name)
	}
	// @@TODO: reenable?
	// refsList = append(refsList, "@refs/heads/master HEAD")

	return refsList, nil
}

func (t *ConscienceTransport) Push() error {
	return nil
}

func (t *ConscienceTransport) NegotiateFetch(r *git.Repository, wants []git.RemoteHead) error {
	s := ""
	for i := range wants {
		s += "  - " + wants[i].Id.String() + " / " + wants[i].Name + "\n"
	}
	fmt.Println(t.repoID, "NegotiateFetch\n", s)
	t.wants = wants
	return nil
}

func (t *ConscienceTransport) DownloadPack(r *git.Repository, progress *git.TransferProgress) error { //, progressCallback C.git_transfer_progress_cb, progressPayload unsafe.Pointer) error {
	fmt.Println(t.repoID, "Download to", r.Path())
	fmt.Printf("  - DownloadPack %+v\n", progress)
	err := t.fetchFromCommit(t.repoID, &repo.Repo{Repository: r}, t.wants[0].Id.String())
	if err != nil {
		return err
	}
	return nil
}

func (t *ConscienceTransport) IsConnected() (bool, error) {
	fmt.Println(t.repoID, "IsConnected")
	return true, nil
}

func (t *ConscienceTransport) Cancel() {
	fmt.Println(t.repoID, "Cancel")
}

func (t *ConscienceTransport) Close() error {
	fmt.Println(t.repoID, "Close")
	t.client.Close()
	return nil
}

func (t *ConscienceTransport) Free() {
	fmt.Println(t.repoID, "Free")
}

func (t *ConscienceTransport) fetchFromCommit(repoID string, r *repo.Repo, commitHashStr string) error {
	commitHashSlice, err := hex.DecodeString(commitHashStr)
	if err != nil {
		return err
	} else if r.HasObject(commitHashSlice) {
		return nil
	}

	commitHash, err := git.NewOid(commitHashStr)
	if err != nil {
		return nil
	}

	// @@TODO: give context a timeout and make it configurable
	ch, uncompressedSize, err := t.client.FetchFromCommit(context.TODO(), repoID, r.Path(), *commitHash)
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
				pw, err := r.PackfileWriter()
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
				packfileCommittedID, err := packfiles[packfileID].Commit()
				if err != nil {
					packfiles[packfileID].Free()
					return errors.WithStack(err)
				}
				packfiles[packfileID].Free()

				fmt.Println("commit packfile", packfileCommittedID.String())

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
	// if env.MachineOutputEnabled {
	//  pw.singleLineWriter.Printf("Progress: %d/%d ", written, total)
	// } else {
	pw.multiLineWriter.Printf(0, "Total:      %v %v/%v = %.02f%%", getProgressBar(written, total), humanize(written), humanize(total), 100*(float64(written)/float64(total)))
	if packfileWritten == packfileTotal {
		pw.multiLineWriter.Printf(pw.lines[packfileID], "pack %v (%v) Done.", packfileID[:6], humanize(packfileTotal))
	} else {
		pw.multiLineWriter.Printf(pw.lines[packfileID], "pack %v %v %v/%v = %.02f%%", packfileID[:6], getProgressBar(packfileWritten, packfileTotal), humanize(packfileWritten), humanize(packfileTotal), 100*(float64(packfileWritten)/float64(packfileTotal)))
	}
	// }
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
