package nodep2p

import (
	"context"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/Conscience/protocol/config"
	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/util"
)

var ErrPolicyMaxBytesExceeded = errors.New("MaxBytes has been exceeded for this replicated repo")

// Handles an incoming request to replicate (pull changes from) a given repository.
func Replicate(ctx context.Context, repoID string, node INode, policy config.ReplicationPolicy, progressCb func(current, total uint64) error) error {
	var maxBytes int64

	// Ensure that the repo has been whitelisted for replication.
	if policy.MaxBytes == 0 {
		return errors.Errorf("replication of repo '%v' not allowed", repoID)
	}

	// Perform the fetch
	{
		r := node.Repo(repoID)
		if r == nil {
			log.Debugf("don't have repo %v locally. cloning.", repoID)

			_, err := node.Clone(ctx, &CloneOptions{
				Node:     node,
				RepoID:   repoID,
				RepoRoot: filepath.Join(node.GetConfig().Node.ReplicationRoot, repoID),
				Bare:     true,
				CheckManifest: func(manifest *Manifest) error {
					if maxBytes >= 0 && manifest.GitObjects.UncompressedSize()+manifest.ChunkObjects.UncompressedSize() > maxBytes {
						return ErrPolicyMaxBytesExceeded
					}
					return nil
				},
				ProgressCb: func(current, total uint64) error {
					if int64(current) > maxBytes {
						return ErrPolicyMaxBytesExceeded
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
			_, err := node.FetchAndSetRef(ctx, &FetchOptions{
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
					if maxBytes > 0 && currentRepoSize+totalDownloadSize > maxBytes {
						return ErrPolicyMaxBytesExceeded
					}
					return nil
				},
				ProgressCb: func(current, total uint64) error {
					if int64(current) > maxBytes {
						return ErrPolicyMaxBytesExceeded
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
