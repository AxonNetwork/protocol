package nodep2p

import (
	"context"
	"path/filepath"
	"time"

	"github.com/pkg/errors"

	netp2p "github.com/libp2p/go-libp2p-net"

	"github.com/Conscience/protocol/config"
	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/util"
)

// Handles an incoming request to replicate (pull changes from) a given repository.
func (h *Host) handleReplicationRequest(stream netp2p.Stream) {
	defer stream.Close()

	log.Printf("[replication server] receiving replication request")

	var req ReplicationRequest
	err := ReadMsg(stream, &req)
	if err != nil {
		log.Errorf("[replication server] error: %v", err)
		return
	}

	// @@TODO: make context timeout configurable
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err = h.Replicate(ctx, req.RepoID, h.Config.Node.ReplicationPolicies[req.RepoID], func(current, total uint64) error {
		return WriteMsg(stream, &Progress{Current: current, Total: total})
	})
	if err != nil {
		log.Errorf("[replication server] error: %v", err)
		err = WriteMsg(stream, &Progress{ErrorMsg: err.Error()})
		if err != nil {
			log.Errorf("[replication server] error: %v", err)
		}
		return
	}

	err = WriteMsg(stream, &Progress{Done: true})
	if err != nil {
		log.Errorf("[replication server] error: %v", err)
		return
	}
}

var ErrPolicyMaxBytesExceeded = errors.New("MaxBytes has been exceeded for this replicated repo")

// Handles an incoming request to replicate (pull changes from) a given repository.
func (h *Host) Replicate(ctx context.Context, repoID string, policy config.ReplicationPolicy, progressCb func(current, total uint64) error) error {
	// Ensure that the repo has been whitelisted for replication.
	if policy.MaxBytes == 0 {
		return errors.Errorf("replication of repo '%v' not allowed", repoID)
	}

	// Perform the fetch
	{
		r := h.repoManager.Repo(repoID)
		if r == nil {
			log.Debugf("don't have repo %v locally. cloning.", repoID)

			_, err := h.Clone(ctx, &CloneOptions{
				RepoID:   repoID,
				RepoRoot: filepath.Join(h.Config.Node.ReplicationRoot, repoID),
				Bare:     policy.Bare,
				CheckManifest: func(manifest *Manifest) error {
					if policy.MaxBytes >= 0 && manifest.GitObjects.UncompressedSize()+manifest.ChunkObjects.UncompressedSize() > policy.MaxBytes {
						return errors.WithStack(ErrPolicyMaxBytesExceeded)
					}
					return nil
				},
				ProgressCb: func(current, total uint64) error {
					if int64(current) > policy.MaxBytes {
						return errors.WithStack(ErrPolicyMaxBytesExceeded)
					}
					if progressCb != nil {
						return progressCb(current, total)
					}
					return nil
				},
			})
			if err != nil {
				return errors.Wrapf(err, "cloning axon://%v remote", repoID)
			}
			log.Debugf("cloned axon://%v remote", repoID)

		} else {
			_, err := h.FetchAndSetRef(ctx, &FetchOptions{
				Repo: r,
				CheckManifest: func(manifest *Manifest) error {
					totalDownloadSize := manifest.GitObjects.UncompressedSize() + manifest.ChunkObjects.UncompressedSize()
					// @@TODO: if we're using DirSize for this (which we should), then we need to
					// make sure replication repos are always bare (otherwise we're unfairly counting
					// each byte twice)
					currentRepoSize, err := util.DirSize(r.Path())
					if err != nil {
						return errors.Wrapf(err, "can't get DirSize( %v ): %v", repoID, r.Path())
					}

					// @@TODO: account for objects we can delete.  note that this will increase the
					// complexity of the entire download process *significantly* given that we can't
					// trust the node we're downloading from to be well-behaved and benevolent.
					if policy.MaxBytes > 0 && currentRepoSize+totalDownloadSize > policy.MaxBytes {
						return errors.WithStack(ErrPolicyMaxBytesExceeded)
					}
					return nil
				},
				ProgressCb: func(current, total uint64) error {
					if int64(current) > policy.MaxBytes {
						return errors.WithStack(ErrPolicyMaxBytesExceeded)
					}
					return nil
				},
			})
			if err != nil {
				return errors.Wrapf(err, "fetching axon:// remote for repo %v", repoID)
			}
		}
	}

	return nil
}
