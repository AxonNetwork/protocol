package nodep2p

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/libgit2/git2go"
	"github.com/pkg/errors"

	"github.com/Conscience/protocol/filters/decode"
	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/repo"
	"github.com/Conscience/protocol/swarm/nodeevents"
)

type CheckManifestCallback func(manifest *Manifest) error

var checkManifestCallback = make(map[*git.Remote]CheckManifestCallback)
var checkManifestCallbackMu = &sync.Mutex{}

func setCheckManifestCallback(remote *git.Remote, cb CheckManifestCallback) {
	checkManifestCallbackMu.Lock()
	defer checkManifestCallbackMu.Unlock()
	if cb != nil {
		checkManifestCallback[remote] = cb
	} else {
		delete(checkManifestCallback, remote)
	}
}

func getCheckManifestCallback(remote *git.Remote) CheckManifestCallback {
	checkManifestCallbackMu.Lock()
	defer checkManifestCallbackMu.Unlock()
	return checkManifestCallback[remote]
}

type CloneOptions struct {
	RepoID        string
	RepoRoot      string
	Bare          bool
	ProgressCb    func(done, total uint64) error
	UserName      string
	UserEmail     string
	CheckManifest CheckManifestCallback
	IsReplication bool
}

func (h *Host) Clone(ctx context.Context, opts *CloneOptions) (*repo.Repo, error) {
	if opts.ProgressCb == nil {
		opts.ProgressCb = func(done, total uint64) error { return nil }
	}

	var innerErr error
	var innerRemote *git.Remote

	if opts.CheckManifest != nil {
		defer func() { setCheckManifestCallback(innerRemote, nil) }()
	}

	cRepo, err := git.Clone("axon://"+opts.RepoID, opts.RepoRoot, &git.CloneOptions{
		Bare: opts.Bare,
		RemoteCreateCallback: func(r *git.Repository, name, url string) (*git.Remote, git.ErrorCode) {
			remote, err := r.Remotes.Create("origin", "axon://"+opts.RepoID)
			if err != nil {
				return nil, git.ErrGeneric
			}

			innerRemote = remote
			if opts.CheckManifest != nil {
				setCheckManifestCallback(remote, opts.CheckManifest)
			}

			return remote, git.ErrOk
		},
		FetchOptions: &git.FetchOptions{
			RemoteCallbacks: git.RemoteCallbacks{
				TransferProgressCallback: func(stats git.TransferProgress) git.ErrorCode {
					select {
					case <-ctx.Done():
						innerErr = errors.WithStack(ctx.Err())
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
		return nil, errors.WithStack(err)
	}

	if opts.UserName != "" {
		err = r.AddUserToConfig(opts.UserName, opts.UserEmail)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	log.Warnf("TRYING TO TRACK REPO <%v>", r.Path())
	r, err = h.repoManager.TrackRepo(r.Path(), true)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if !opts.Bare {
		err = decodeFiles(r.Path())
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	return r, nil
}

type PushOptions struct {
	Repo       *repo.Repo
	BranchName string
	Force      bool
	ProgressCb func(percent int)
}

var ErrRequiresForcePush = errors.New("requires force push")
var ErrRepoIDNotRegistered = errors.New("repo ID not registered")

func (h *Host) Push(superCtx context.Context, opts *PushOptions) (string, error) {
	r := opts.Repo

	repoID, err := r.RepoID()
	if err != nil {
		return "", err
	}

	// @@TODO: make context timeout configurable
	ctx, cancel := context.WithTimeout(superCtx, 15*time.Second)
	defer cancel()
	isRegistered, err := h.ethClient.IsRepoIDRegistered(ctx, repoID)
	if !isRegistered {
		return "", errors.WithStack(ErrRepoIDNotRegistered)
	}

	ref, err := r.References.Dwim(opts.BranchName)
	if err != nil {
		return "", errors.WithStack(err)
	}
	branch := ref.Branch()

	// branch, err := r.LookupBranch(opts.BranchName, git.BranchLocal)
	// if err != nil {
	// 	return "", errors.WithStack(err)
	// }

	srcRef, err := branch.Reference.Resolve()
	if err != nil {
		return "", errors.WithStack(err)
	}

	localCommitOid := srcRef.Target()

	// Check to make sure that the new commit we're pushing is a descendant of the commit in the
	// remote.  If not, the user must specify opts.Force or the push will fail.
	var currentCommitOid git.Oid
	{
		// @@TODO: make context timeout configurable
		ctx, cancel := context.WithTimeout(superCtx, 15*time.Second)
		defer cancel()

		currentCommitOid, err = h.ethClient.GetRemoteRef(ctx, repoID, branch.Reference.Name())
		if err != nil {
			return "", errors.WithStack(err)
		}

		var isDescendant bool
		if currentCommitOid.IsZero() {
			// nothing pushed to contract yet
			isDescendant = true
		} else {
			isDescendant, err = r.DescendantOf(localCommitOid, &currentCommitOid)
			if err != nil {
				return "", errors.WithStack(err)
			}
		}

		if !isDescendant && !opts.Force {
			return "", errors.WithStack(ErrRequiresForcePush)
		}
	}

	// @@TODO: make context timeout configurable
	ctx, cancel = context.WithTimeout(superCtx, 15*time.Second)
	defer cancel()

	// Tell the node to announce the new content so that replicator nodes can find and pull it.
	err = h.AnnounceRepo(ctx, repoID)
	if err != nil {
		return "", errors.WithStack(err)
	}

	// @@TODO: make context timeout configurable
	ctx, cancel = context.WithTimeout(superCtx, 15*time.Second)
	defer cancel()

	tx, err := h.ethClient.UpdateRemoteRef(ctx, repoID, branch.Reference.Name(), currentCommitOid, *localCommitOid)
	if err != nil {
		return "", errors.WithStack(err)
	}

	txResult := <-tx.Await(ctx)
	if txResult.Err != nil {
		return "", errors.WithStack(txResult.Err)
	} else if txResult.Receipt.Status == 0 {
		return "", errors.New("transaction failed")
	}

	// @@TODO: make context timeout configurable
	ctx, cancel = context.WithTimeout(superCtx, 60*time.Second)
	defer cancel()

	ch, err := h.RequestReplication(ctx, repoID)
	if err != nil {
		return "", err
	}

	for progress := range ch {
		if progress.Error != nil {
			return "", errors.WithStack(progress.Error)
		}
		opts.ProgressCb(int(progress.Current))
	}

	h.eventBus.NotifyWatchers(nodeevents.MaybeEvent{
		EventType: nodeevents.EventType_PushedRepo,
		PushedRepoEvent: &nodeevents.PushedRepoEvent{
			RepoID:     repoID,
			RepoRoot:   r.Path(),
			BranchName: opts.BranchName,
			Commit:     localCommitOid.String(),
		},
	})

	return localCommitOid.String(), nil
}

type FetchOptions struct {
	Repo          *repo.Repo
	ProgressCb    func(done, total uint64) error
	CheckManifest CheckManifestCallback
	EventBus      *nodeevents.EventBus
}

// Perform a fetch on the first Axon remote found in the given repo's config.
func (h *Host) fetchAxonRemote(ctx context.Context, opts *FetchOptions) ([]string, error) {
	if opts.ProgressCb == nil {
		opts.ProgressCb = func(done, total uint64) error { return nil }
	}

	remote, err := opts.Repo.AxonRemote()
	if err != nil {
		return nil, err
	}

	setCheckManifestCallback(remote, opts.CheckManifest)
	defer setCheckManifestCallback(remote, nil)

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
					innerErr = errors.WithStack(ctx.Err())
					return git.ErrGeneric
				default:
				}
				if opts.ProgressCb != nil {
					innerErr = opts.ProgressCb(uint64(stats.ReceivedObjects), uint64(stats.TotalObjects))
					if innerErr != nil {
						return git.ErrGeneric
					}
				}
				return git.ErrOk
			},
		},
	}, "")
	if innerErr != nil {
		return nil, innerErr
	} else if err != nil {
		return nil, errors.WithStack(err)
	}
	return updatedRefs, nil
}

// Fetches refs and objects from an Axon remote and then updates local refs that are tracking those
// remote refs.
func (h *Host) FetchAndSetRef(ctx context.Context, opts *FetchOptions) ([]string, error) {
	updatedRefs, err := h.fetchAxonRemote(ctx, opts)
	if err != nil {
		return nil, err
	}

	// @@TODO: don't assume that a local ref is tracking a remote simply because of its name.  Check
	// the .git/config setup first.
	repo := opts.Repo
	for _, name := range updatedRefs {
		if strings.HasPrefix(name, "refs/remotes/origin") { // @@TODO: don't hardcode
			ref, err := repo.References.Lookup(name)
			if err != nil {
				return nil, errors.WithStack(err)
			}

			oid := ref.Target()
			localRefName := strings.Replace(name, "refs/remotes/origin", "refs/heads", 1) // @@TODO: don't hardcode
			_, err = repo.References.Create(localRefName, oid, true, "")
			if err != nil {
				return nil, errors.WithStack(err)
			}
		}
	}

	repoID, err := opts.Repo.RepoID()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	h.eventBus.NotifyWatchers(nodeevents.MaybeEvent{
		EventType: nodeevents.EventType_PulledRepo,
		PulledRepoEvent: &nodeevents.PulledRepoEvent{
			RepoID:      repoID,
			RepoRoot:    opts.Repo.Path(),
			UpdatedRefs: updatedRefs,
		},
	})

	return updatedRefs, nil
}

type PullOptions struct {
	Repo       *repo.Repo
	RemoteName string
	BranchName string
	ProgressCb func(done, total uint64) error
}

func (h *Host) Pull(ctx context.Context, opts *PullOptions) ([]string, error) {
	var err error

	r := opts.Repo

	// 1. stash worktree
	{

		name, email, err := r.UserIdentityFromConfig()
		if err != nil {
			name = ""
			email = ""
			return nil, errors.WithStack(err)
		}

		sig := &git.Signature{
			Name:  name,
			Email: email,
			When:  time.Now(),
		}

		var didStash bool

		_, err = r.Repository.Stashes.Save(sig, "", git.StashDefault)
		if err != nil && strings.Contains(err.Error(), "there is nothing to stash") {
			// no-op
		} else if err != nil {
			return nil, errors.WithStack(err)
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
					err = errors.WithStack(err2)
				}
				return
			}

			stashApplyOpts.CheckoutOptions.Strategy |= git.CheckoutAllowConflicts | git.CheckoutConflictStyleMerge | git.CheckoutDontOverwriteIgnored

			err2 = r.Repository.Stashes.Pop(0, stashApplyOpts)
			if err2 != nil {
				log.Errorln("repo.Pull: error popping stash:", err2)
				if err == nil {
					err = errors.WithStack(err2)
				}
			}
		}()
	}

	// 2. fetch
	var updatedRefs []string
	var remote *git.Remote
	{
		remote, err = r.Remotes.Lookup(opts.RemoteName)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		var innerErr error
		err = remote.Fetch([]string{}, &git.FetchOptions{
			RemoteCallbacks: git.RemoteCallbacks{
				UpdateTipsCallback: func(refname string, a *git.Oid, b *git.Oid) git.ErrorCode {
					updatedRefs = append(updatedRefs, refname)
					return git.ErrOk
				},
				TransferProgressCallback: func(stats git.TransferProgress) git.ErrorCode {
					select {
					case <-ctx.Done():
						innerErr = errors.WithStack(ctx.Err())
						return git.ErrGeneric
					default:
					}
					if opts.ProgressCb != nil {
						innerErr = opts.ProgressCb(uint64(stats.ReceivedObjects), uint64(stats.TotalObjects))
						if innerErr != nil {
							return git.ErrGeneric
						}
					}
					return git.ErrOk
				},
			},
		}, "")
		if innerErr != nil {
			return nil, innerErr
		} else if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	// 3. merge
	{
		if r.State() != git.RepositoryStateNone {
			return nil, errors.Errorf("repository in unexpected state prior to merge: %v", r.State())
		}

		remoteBranch, err := r.LookupBranch(opts.RemoteName+"/"+opts.BranchName, git.BranchRemote)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		mergeHead, err := r.AnnotatedCommitFromRef(remoteBranch.Reference)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		incomingHeads := []*git.AnnotatedCommit{mergeHead}
		analysis, preference, err := r.MergeAnalysis(incomingHeads)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		if analysis&git.MergeAnalysisUpToDate > 0 {
			// Already up to date.

			return updatedRefs, nil

		} else if analysis&git.MergeAnalysisUnborn > 0 ||
			(analysis&git.MergeAnalysisFastForward > 0 && preference&git.MergePreferenceNoFastForward == 0) {
			// Fast-forward merge.

			unborn := analysis&git.MergeAnalysisUnborn > 0
			err = doFastForward(r, remoteBranch.Target(), unborn)
			if err != nil {
				return nil, errors.WithStack(err)
			} else {
				return updatedRefs, nil
			}

		} else if analysis&git.MergeAnalysisNormal > 0 {
			// Regular merge.

			mergeOpts, err := git.DefaultMergeOptions()
			if err != nil {
				return nil, errors.WithStack(err)
			}
			mergeOpts.TreeFlags = git.MergeTreeFindRenames

			err = r.Merge(incomingHeads, &mergeOpts, &git.CheckoutOpts{Strategy: git.CheckoutForce | git.CheckoutAllowConflicts})
			if err != nil {
				return nil, errors.WithStack(err)
			}
		}

		index, err := r.Index()
		if err != nil {
			return nil, errors.WithStack(err)
		}

		if index.HasConflicts() == false {
			err = createMergeCommit(r, index, remote, remoteBranch)
			if err != nil {
				return nil, errors.WithStack(err)
			}
		}
	}

	err = r.StateCleanup()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = decodeFiles(r.Path())

	repoID, err := r.RepoID()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	h.eventBus.NotifyWatchers(nodeevents.MaybeEvent{
		EventType: nodeevents.EventType_PulledRepo,
		PulledRepoEvent: &nodeevents.PulledRepoEvent{
			RepoID:      repoID,
			RepoRoot:    r.Path(),
			UpdatedRefs: updatedRefs,
		},
	})

	return updatedRefs, nil
}

func createMergeCommit(r *repo.Repo, index *git.Index, remote *git.Remote, remoteBranch *git.Branch) error {
	headRef, err := r.Head()
	if err != nil {
		return errors.WithStack(err)
	}

	parentObjOne, err := headRef.Peel(git.ObjectCommit)
	if err != nil {
		return errors.WithStack(err)
	}

	parentObjTwo, err := remoteBranch.Reference.Peel(git.ObjectCommit)
	if err != nil {
		return errors.WithStack(err)
	}

	parentCommitOne, err := parentObjOne.AsCommit()
	if err != nil {
		return errors.WithStack(err)
	}

	parentCommitTwo, err := parentObjTwo.AsCommit()
	if err != nil {
		return errors.WithStack(err)
	}

	treeOid, err := index.WriteTree()
	if err != nil {
		return errors.WithStack(err)
	}

	tree, err := r.LookupTree(treeOid)
	if err != nil {
		return errors.WithStack(err)
	}

	remoteBranchName, err := remoteBranch.Name()
	if err != nil {
		return errors.WithStack(err)
	}

	userName, userEmail, err := r.UserIdentityFromConfig()
	if err != nil {
		userName = ""
		userEmail = ""
	}

	var (
		now       = time.Now()
		author    = &git.Signature{Name: userName, Email: userEmail, When: now}
		committer = &git.Signature{Name: userName, Email: userEmail, When: now}
		message   = fmt.Sprintf(`Merge branch '%v' of %v`, remoteBranchName, remote.Url())
		parents   = []*git.Commit{
			parentCommitOne,
			parentCommitTwo,
		}
	)

	_, err = r.CreateCommit(headRef.Name(), author, committer, message, tree, parents...)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

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

func decodeFiles(repoRoot string) error {
	r, err := repo.Open(repoRoot)
	if err != nil {
		return err
	}

	head, err := r.Head()
	if err != nil {
		// no head
		return nil
	}

	commitObj, err := head.Peel(git.ObjectCommit)
	if err != nil {
		return err
	}

	commit, err := commitObj.AsCommit()
	if err != nil {
		return err
	}

	tree, err := commit.Tree()
	if err != nil {
		return err
	}

	odb, err := r.Odb()
	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	waitCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitCh)
	}()

	errCh := make(chan error)
	var innerErr error
	err = tree.Walk(func(relPath string, entry *git.TreeEntry) int {
		isChunked, err := r.FileIsChunked(entry.Name, commitObj.Id())
		if err != nil {
			innerErr = err
			return -1
		}

		if isChunked {
			go func() {
				wg.Add(1)
				err = decodeFile(r, relPath, entry, odb)
				if err != nil {
					errCh <- err
				}
				wg.Done()
			}()
		}
		return 0
	})
	if innerErr != nil {
		return innerErr
	} else if err != nil {
		return err
	}

	select {
	case msg := <-errCh:
		return msg
	case <-waitCh:
		return nil
	}

	return nil
}

func decodeFile(r *repo.Repo, relPath string, entry *git.TreeEntry, odb *git.Odb) error {
	odbObj, err := odb.Read(entry.Id)
	if err != nil {
		return err
	}
	defer odbObj.Free()

	data := odbObj.Data()
	length := int(odbObj.Len())
	if length%65 != 0 {
		return errors.Errorf("invalid axon object: hash lengths not parsable")
	}
	rPipe, wPipe := io.Pipe()
	go func() {
		defer wPipe.Close()
		for i := 0; i < length; i += 65 {
			n, err := wPipe.Write(data[i : i+65])
			if err != nil {
				wPipe.CloseWithError(err)
				break
			} else if n < 64 {
				wPipe.CloseWithError(errors.Errorf("did not write full object"))
				break
			}
		}
	}()

	p := filepath.Join(r.Path(), relPath, entry.Name)
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	defer f.Close()

	missingChunks := false
	fileReader := decode.Decode(r, rPipe, func(chunks [][]byte) error {
		// chunks should have already been pulled
		// if missing, the user did a sparse checkout, and we write empty files
		missingChunks = true
		return nil
	})
	defer fileReader.Close()

	if missingChunks {
		return nil
	}

	_, err = io.Copy(f, fileReader)
	if err != nil {
		return err
	}

	err = f.Close()
	if err != nil {
		return err
	}

	return nil
}
