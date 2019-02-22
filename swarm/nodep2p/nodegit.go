package nodep2p

import (
	"context"
	"strings"
	"time"

	"github.com/libgit2/git2go"
	"github.com/pkg/errors"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/repo"
	"github.com/Conscience/protocol/swarm/nodeeth"
)

type CloneOptions struct {
	RepoID     string
	RepoRoot   string
	Bare       bool
	ProgressCb func(done, total uint64) error
	UserName   string
	UserEmail  string
}

func Clone(ctx context.Context, opts *CloneOptions) (*repo.Repo, error) {
	if opts.ProgressCb == nil {
		opts.ProgressCb = func(done, total uint64) error { return nil }
	}

	var innerErr error
	cRepo, err := git.Clone("conscience://"+opts.RepoID, opts.RepoRoot, &git.CloneOptions{
		Bare: opts.Bare,
		FetchOptions: &git.FetchOptions{
			RemoteCallbacks: git.RemoteCallbacks{
				TransferProgressCallback: func(stats git.TransferProgress) git.ErrorCode {
					select {
					case <-ctx.Done():
						innerErr = ctx.Err()
						return git.ErrGeneric
					default:
					}

					innerErr = opts.ProgressCb(uint64(stats.ReceivedObjects), uint64(stats.TotalObjects))
					if innerErr != nil {
						return git.ErrGeneric
					}
					return git.ErrOk
				},
			},
		},
	})
	if innerErr != nil {
		return nil, innerErr
	} else if err != nil {
		return nil, err
	}

	r := &repo.Repo{Repository: cRepo}

	err = r.SetupConfig(opts.RepoID)
	if err != nil {
		return nil, err
	}

	if opts.UserName != "" {
		err = r.AddUserToConfig(opts.UserName, opts.UserEmail)
		if err != nil {
			return nil, err
		}
	}

	return r, nil
}

type PushOptions struct {
	Node interface {
		AnnounceRepo(ctx context.Context, repoID string) error
		UpdateRef(ctx context.Context, repoID string, branchRefName string, commitID string) (*nodeeth.Transaction, error)
		RequestReplication(ctx context.Context, repoID string) <-chan MaybeReplProgress
	}
	Repo       *repo.Repo
	BranchName string
	ProgressCb func(percent int)
}

func Push(ctx context.Context, opts *PushOptions) error {
	// Tell the node to announce the new content so that replicator nodes can find and pull it.
	// @@TODO: make context timeout configurable
	ctx1, cancel1 := context.WithTimeout(ctx, 15*time.Second)
	defer cancel1()

	repoID, err := opts.Repo.RepoID()
	if err != nil {
		return err
	}

	err = opts.Node.AnnounceRepo(ctx1, repoID)
	if err != nil {
		return err
	}

	branch, err := opts.Repo.LookupBranch(opts.BranchName, git.BranchLocal)
	if err != nil {
		return err
	}

	srcRef, err := branch.Reference.Resolve()
	if err != nil {
		return err
	}

	commitOid := srcRef.Target()

	// @@TODO: make context timeout configurable
	ctx2, cancel2 := context.WithTimeout(ctx, 15*time.Second)
	defer cancel2()

	tx, err := opts.Node.UpdateRef(ctx2, repoID, branch.Reference.Name(), commitOid.String())
	if err != nil {
		return err
	}

	txResult := <-tx.Await(ctx)
	if txResult.Err != nil {
		return txResult.Err
	} else if txResult.Receipt.Status == 0 {
		return errors.New("transaction failed")
	}

	// @@TODO: make context timeout configurable
	ctx3, cancel3 := context.WithTimeout(ctx, 60*time.Second)
	defer cancel3()

	ch := opts.Node.RequestReplication(ctx3, repoID)
	for progress := range ch {
		if progress.Error != nil {
			return progress.Error
		}
		opts.ProgressCb(progress.Percent)
	}

	return nil
}

type FetchOptions struct {
	Repo       *repo.Repo
	ProgressCb func(done, total uint64) error
}

func FetchConscienceRemote(ctx context.Context, opts *FetchOptions) ([]string, error) {
	if opts.ProgressCb == nil {
		opts.ProgressCb = func(done, total uint64) error { return nil }
	}

	remote, err := opts.Repo.ConscienceRemote()
	if err != nil {
		return nil, err
	}

	var innerErr error
	var updatedRefs []string
	err = remote.Fetch([]string{}, &git.FetchOptions{
		RemoteCallbacks: git.RemoteCallbacks{
			UpdateTipsCallback: func(refname string, a *git.Oid, b *git.Oid) git.ErrorCode {
				updatedRefs = append(updatedRefs, refname)
				return git.ErrOk
			},
			TransferProgressCallback: func(stats git.TransferProgress) git.ErrorCode {
				select {
				case <-ctx.Done():
					innerErr = ctx.Err()
					return git.ErrGeneric
				default:
				}

				innerErr = opts.ProgressCb(uint64(stats.ReceivedObjects), uint64(stats.TotalObjects))
				if innerErr != nil {
					return git.ErrGeneric
				}
				return git.ErrOk
			},
		},
	}, "")
	if innerErr != nil {
		return nil, innerErr
	} else if err != nil {
		return nil, err
	}
	return updatedRefs, nil
}

func GetFetchRefspecs(remote *git.Remote) []*git.Refspec {
	refspecs := []*git.Refspec{}

	n := remote.RefspecCount()
	for i := uint(0); i < n; i++ {
		refspec := remote.GetRefspec(i)
		if refspec.Direction() == git.ConnectDirectionFetch {
			refspecs = append(refspecs, refspec)
		}
	}
	return refspecs
}

type PullOptions struct {
	Repo       *repo.Repo
	Remote     string
	Branch     string
	ProgressCb func(done, total uint64) error
}

func Pull(ctx context.Context, opts *PullOptions) (err error) {
	// 1. stash worktree
	{
		cfg, err := opts.Repo.Config()
		if err != nil {
			return err
		}

		name, err := cfg.LookupString("user.name")
		if err != nil {
			return err
		}

		email, err := cfg.LookupString("user.email")
		if err != nil {
			return err
		}

		sig := &git.Signature{
			Name:  name,
			Email: email,
			When:  time.Now(),
		}

		var didStash bool

		_, err = opts.Repo.Repository.Stashes.Save(sig, "", git.StashDefault)
		if err != nil && strings.Contains(err.Error(), "there is nothing to stash") {
			// no-op
		} else if err != nil {
			return err
		} else {
			didStash = true
		}

		// Pop the stash when we're done
		defer func() {
			if !didStash {
				return
			}

			stashApplyOpts, err2 := git.DefaultStashApplyOptions()
			if err2 != nil {
				log.Errorln("repo.Pull: could not create git.DefaultStashApplyOptions:", err2)
				if err == nil {
					err = err2
				}
				return
			}

			stashApplyOpts.CheckoutOptions.Strategy |= git.CheckoutAllowConflicts | git.CheckoutConflictStyleMerge | git.CheckoutDontOverwriteIgnored

			err2 = opts.Repo.Repository.Stashes.Pop(0, stashApplyOpts)
			if err2 != nil {
				log.Errorln("repo.Pull: error popping stash:", err2)
				if err == nil {
					err = err2
				}
			}
		}()
	}

	// 2. fetch
	{
		remote, err := opts.Repo.Remotes.Lookup(opts.Remote)
		if err != nil {
			return err
		}

		var innerErr error
		err = remote.Fetch([]string{}, &git.FetchOptions{
			RemoteCallbacks: git.RemoteCallbacks{
				TransferProgressCallback: func(stats git.TransferProgress) git.ErrorCode {
					select {
					case <-ctx.Done():
						innerErr = ctx.Err()
						return git.ErrGeneric
					default:
					}

					innerErr = opts.ProgressCb(uint64(stats.ReceivedObjects), uint64(stats.TotalObjects))
					if innerErr != nil {
						return git.ErrGeneric
					}
					return git.ErrOk
				},
			},
		}, "")
		if innerErr != nil {
			return innerErr
		} else if err != nil {
			return err
		}
	}

	// 3. merge
	{
		if opts.Repo.State() != git.RepositoryStateNone {
			return errors.Errorf("repository in unexpected state prior to merge: %v", opts.Repo.State())
		}

		branch, err := opts.Repo.LookupBranch(opts.Remote+"/"+opts.Branch, git.BranchRemote)
		if err != nil {
			return err
		}

		mergeHead, err := opts.Repo.AnnotatedCommitFromRef(branch.Reference)
		if err != nil {
			return err
		}

		incomingHeads := []*git.AnnotatedCommit{mergeHead}
		analysis, preference, err := opts.Repo.MergeAnalysis(incomingHeads)
		if err != nil {
			return err
		}

		if analysis&git.MergeAnalysisUpToDate > 0 {
			// Already up to date.

			return nil

		} else if analysis&git.MergeAnalysisUnborn > 0 ||
			(analysis&git.MergeAnalysisFastForward > 0 && preference&git.MergePreferenceNoFastForward == 0) {
			// Fast-forward merge.

			unborn := analysis&git.MergeAnalysisUnborn > 0
			return doFastForward(opts.Repo, branch.Target(), unborn)

		} else if analysis&git.MergeAnalysisNormal > 0 {
			// Regular merge.

			mergeOpts, err := git.DefaultMergeOptions()
			if err != nil {
				return err
			}
			mergeOpts.TreeFlags = git.MergeTreeFindRenames

			err = opts.Repo.Merge(incomingHeads, &mergeOpts, &git.CheckoutOpts{Strategy: git.CheckoutForce | git.CheckoutAllowConflicts})
			if err != nil {
				return err
			}
		}

		index, err := opts.Repo.Index()
		if err != nil {
			return err
		}

		if index.HasConflicts() == false {

		}
	}
	return nil
}

// func createMergeCommit(r *repo.Repo, index *git.Index, mergeOpts *git.MergeOpts) error {

// }

func doFastForward(r *repo.Repo, targetOid *git.Oid, unborn bool) error {
	var targetRef *git.Reference

	if unborn {
		headRef, err := r.References.Lookup("HEAD")
		if err != nil {
			return err
		}

		symref := headRef.SymbolicTarget()

		targetRef, err = r.References.Create(symref, targetOid, false, "")
		if err != nil {
			return err
		}

	} else {
		var err error
		targetRef, err = r.Head()
		if err != nil {
			return err
		}
	}

	commit, err := r.LookupCommit(targetOid)
	if err != nil {
		return err
	}

	commitTree, err := commit.Tree()
	if err != nil {
		return err
	}

	err = r.CheckoutTree(commitTree, &git.CheckoutOpts{Strategy: git.CheckoutSafe})
	if err != nil {
		return err
	}

	_, err = targetRef.SetTarget(targetOid, "")
	return err
}
