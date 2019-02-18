package repo

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/libgit2/git2go"
	"github.com/pkg/errors"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/util"
)

type Repo struct {
	*git.Repository
}

const (
	CONSCIENCE_DATA_SUBDIR = "data"
	CONSCIENCE_HASH_LENGTH = 32
	GIT_HASH_LENGTH        = 20
)

var (
	Err404 = errors.New("not found")
)

func EnsureExists(path string) (*Repo, error) {
	r, err := Open(path)
	if err == nil {
		return r, nil
	} else if errors.Cause(err) != Err404 {
		return nil, err
	}
	return Init(path)
}

func Init(path string) (*Repo, error) {
	gitRepo, err := git.InitRepository(path, false)
	if err != nil {
		return nil, errors.Wrapf(err, "could not initialize repo at path '%v'", path)
	}

	return &Repo{Repository: gitRepo}, nil
}

func Open(path string) (*Repo, error) {
	gitRepo, err := git.OpenRepository(path)
	if err != nil && strings.Contains(err.Error(), "failed to resolve path") {
		return nil, errors.WithStack(Err404)
	} else if err != nil {
		return nil, errors.WithStack(err)
	}

	return &Repo{Repository: gitRepo}, nil
}

func (r *Repo) Path() string {
	p := strings.Replace(r.Repository.Path(), "/.git/", "", -1)
	fmt.Println("PATH PATH PATH", p)
	return p
}

func (r *Repo) RepoID() (string, error) {
	cfg, err := r.Config()
	if err != nil {
		return "", errors.Wrapf(err, "could not open repo config at path '%v'", r.Path())
	}
	defer cfg.Free()

	repoID, err := cfg.LookupString("conscience.repoid")
	if err != nil {
		return "", errors.Wrapf(err, "error looking up conscience.repoid in .git/config (path: %v)", r.Path())
	}

	return repoID, nil
}

// Returns true if the object is known, false otherwise.
func (r *Repo) HasObject(objectID []byte) bool {
	if len(objectID) == CONSCIENCE_HASH_LENGTH {
		p := filepath.Join(r.Path(), ".git", CONSCIENCE_DATA_SUBDIR, hex.EncodeToString(objectID))
		_, err := os.Stat(p)
		return err == nil || !os.IsNotExist(err)

	} else if len(objectID) == GIT_HASH_LENGTH {
		x, err := r.Lookup(util.OidFromBytes(objectID))
		if err == nil {
			x.Free()
		}
		return err == nil
	}

	return false
}

// Open an object for reading.  It is the caller's responsibility to .Close() the object when finished.
func (r *Repo) OpenObject(objectID []byte) (ObjectReader, error) {
	if len(objectID) == CONSCIENCE_HASH_LENGTH {
		// Open a Conscience object
		p := filepath.Join(r.Path(), ".git", CONSCIENCE_DATA_SUBDIR, hex.EncodeToString(objectID))

		f, err := os.Open(p)
		if os.IsNotExist(err) {
			return nil, errors.WithStack(Err404)
		} else if err != nil {
			return nil, errors.WithStack(err)
		}

		stat, err := f.Stat()
		if err != nil {
			return nil, errors.Wrapf(err, "could not stat file '%v'", p)
		}

		or := &objectReader{
			Reader:     f,
			Closer:     f,
			objectType: 0,
			objectLen:  uint64(stat.Size()),
		}
		return or, nil

	} else if len(objectID) == GIT_HASH_LENGTH {
		odb, err := r.Odb()
		if err != nil {
			return nil, errors.WithStack(err)
		}

		stream, err := odb.NewReadStream(util.OidFromBytes(objectID))
		if err != nil {
			return nil, errors.WithStack(err)
		}

		or := &objectReader{
			Reader: stream,
			Closer: FuncCloser(func() error {
				stream.Close()
				stream.Free()
				return nil
			}),
			objectType: stream.Type,
			objectLen:  stream.Size,
		}
		return or, nil

	} else {
		return nil, errors.Errorf("objectID is wrong size (%v)", len(objectID))
	}
}

func (r *Repo) OpenFileInWorktree(filename string) (ObjectReader, error) {
	f, err := os.Open(filepath.Join(r.Path(), filename))
	if os.IsNotExist(err) {
		return nil, errors.WithStack(Err404)
	} else if err != nil {
		return nil, errors.WithStack(err)
	}

	stat, err := f.Stat()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &objectReader{
		Reader:    f,
		Closer:    f,
		objectLen: uint64(stat.Size()),
		// objectType: ,
	}, nil
}

func (r *Repo) OpenFileAtCommit(filename string, commitID CommitID) (ObjectReader, error) {
	commit, err := r.ResolveCommit(commitID)
	if err != nil {
		return nil, err
	}
	defer commit.Free()

	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}
	defer tree.Free()

	treeEntry := tree.EntryByName(filename)
	if treeEntry == nil {
		return nil, errors.WithStack(Err404)
	}

	return r.OpenObject(treeEntry.Id[:])
}

func (r *Repo) ResolveCommitHash(commitID CommitID) (git.Oid, error) {
	if commitID.Ref != "" {
		ref, err := r.References.Lookup(commitID.Ref)
		if err != nil {
			return git.Oid{}, err
		}
		defer ref.Free()

		ref, err = ref.Resolve()
		if err != nil {
			return git.Oid{}, err
		}
		defer ref.Free()

		oid := ref.Target()
		if oid == nil {
			return git.Oid{}, errors.Errorf("could not resolve commit ref '%s' to a revision", commitID.Ref)
		}
		return *oid, nil

	} else if commitID.Hash != nil && !commitID.Hash.IsZero() {
		return *commitID.Hash, nil

	} else {
		return git.Oid{}, errors.Errorf("must specify commit hash or commit ref")
	}
}

func (r *Repo) ResolveCommit(commitID CommitID) (*git.Commit, error) {
	oid, err := r.ResolveCommitHash(commitID)
	if err != nil {
		return nil, err
	}
	return r.LookupCommit(&oid)
}

func (r *Repo) SetupConfig(repoID string) error {
	cfg, err := r.Config()
	if err != nil {
		return errors.Wrapf(err, "could not get repo config (repoID: %v, path: %v)", repoID, r.Path())
	}
	defer cfg.Free()

	_repoID, err := cfg.LookupString("conscience.repoid")
	if err != nil || _repoID != repoID {
		err = cfg.SetString("conscience.repoid", repoID)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	cleanFilter, err := cfg.LookupString("filter.conscience.clean")
	if err != nil || cleanFilter != "conscience_encode" {
		err = cfg.SetString("filter.conscience.clean", "conscience_encode")
		if err != nil {
			return errors.WithStack(err)
		}
	}

	smudgeFilter, err := cfg.LookupString("filter.conscience.smudge")
	if err != nil || smudgeFilter != "conscience_encode" {
		err = cfg.SetString("filter.conscience.smudge", "conscience_decode")
		if err != nil {
			return errors.WithStack(err)
		}
	}

	// Check the remotes
	{
		remoteNames, err := r.Remotes.List()
		if err != nil {
			return errors.WithStack(err)
		}

		found := false
		for _, remoteName := range remoteNames {
			remote, err := r.Remotes.Lookup(remoteName)
			if err != nil {
				return errors.WithStack(err)
			}

			url := remote.Url()
			remote.Free()

			if url == "conscience://"+repoID {
				found = true
				break
			}
		}

		if !found {
			remoteName := "origin"

			remote, err := r.Remotes.Lookup("origin")
			if err == nil {
				remote.Free()
				// Already has an 'origin' remote, so we use the repoID instead.
				// @@TODO: what if this remote name already exists too?
				remoteName = repoID
			}

			remote, err = r.Remotes.CreateWithFetchspec(remoteName, "conscience://"+repoID, "+refs/heads/*:refs/remotes/"+remoteName+"/*")
			if err != nil {
				return errors.Wrapf(err, "could not create remote (repoID: %v, path: %v)", repoID, r.Path())
			}
			remote.Free()
		}
	}

	return nil
}

func (r *Repo) AddUserToConfig(name string, email string) error {
	cfg, err := r.Config()
	if err != nil {
		return errors.Wrapf(err, "could not get repo config (path: %v)", r.Path())
	}
	defer cfg.Free()

	if len(name) > 0 {
		err = cfg.SetString("user.name", name)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	if len(email) > 0 {
		err = cfg.SetString("user.email", email)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

type PackfileWriter interface {
	io.Writer
	Commit() (*git.Oid, error)
	Free()
}

// @@TODO: try implementing https://github.com/libgit2/git2go/pull/416
func (r *Repo) PackfileWriter() (PackfileWriter, error) {
	odb, err := r.Odb()
	if err != nil {
		return nil, err
	}

	return git.NewIndexer(filepath.Join(r.Path(), ".git", "objects", "pack"), odb, func(stats git.TransferProgress) git.ErrorCode {
		fmt.Printf("stats = %+v\n", stats)
		return git.ErrOk
	})
}

func (r *Repo) ListFiles(ctx context.Context, commitID CommitID) ([]File, error) {
	if commitID.Ref == "working" {
		return r.listFilesWorktree(ctx)
	} else {
		return r.listFilesCommit(ctx, commitID)
	}
}

// Returns the file list for the current worktree.
func (r *Repo) listFilesWorktree(ctx context.Context) ([]File, error) {
	statusList, err := r.StatusList(&git.StatusOptions{
		Flags: git.StatusOptIncludeUntracked | git.StatusOptIncludeUnmodified | git.StatusOptRecurseUntrackedDirs | git.StatusOptRenamesHeadToIndex | git.StatusOptRenamesIndexToWorkdir,
		Show:  git.StatusShowIndexAndWorkdir,
	})
	if err != nil {
		return nil, err
	}
	defer statusList.Free()

	n, err := statusList.EntryCount()
	if err != nil {
		return nil, err
	}

	files := make([]File, n)
	for i := 0; i < n; i++ {
		entry, err := statusList.ByIndex(i)
		if err != nil {
			return nil, err
		}

		files[i] = mapStatusEntry(entry)
	}
	return files, nil
}

// Simplifies the interpretation of 'status' for a UI that primarily needs to display information
// about files in the worktree
func mapStatusEntry(entry git.StatusEntry) File {
	// Notes:
	// - IndexToWorkdir.NewFile.Oid is ~always~ empty, presumably because by definition it means "file that isn't in the object DB yet."

	var file File
	file.Filename = entry.IndexToWorkdir.NewFile.Path

	if (entry.Status & git.StatusIndexNew) > 0 {
		file.Filename = entry.HeadToIndex.NewFile.Path
		file.Status.Staged = '?'

	} else if (entry.Status & git.StatusIndexModified) > 0 {
		file.Filename = entry.HeadToIndex.NewFile.Path
		file.Status.Staged = 'M'

	} else if (entry.Status & git.StatusIndexDeleted) > 0 {
		file.Filename = entry.HeadToIndex.NewFile.Path
		file.Status.Staged = 'D'

	} else if (entry.Status & git.StatusIndexRenamed) > 0 {
		file.Filename = entry.HeadToIndex.NewFile.Path
		file.Status.Staged = 'M'

	} else if (entry.Status & git.StatusIndexTypeChange) > 0 {
		file.Filename = entry.HeadToIndex.NewFile.Path
		file.Status.Staged = 'M'
	}

	// ----

	if (entry.Status & git.StatusWtNew) > 0 {
		file.Filename = entry.IndexToWorkdir.NewFile.Path
		file.Status.Unstaged = '?'

	} else if (entry.Status & git.StatusWtModified) > 0 {
		file.Filename = entry.IndexToWorkdir.NewFile.Path
		file.Status.Unstaged = 'M'
		file.Hash = *entry.IndexToWorkdir.OldFile.Oid

	} else if (entry.Status & git.StatusWtDeleted) > 0 {
		file.Filename = entry.IndexToWorkdir.NewFile.Path
		file.Status.Unstaged = 'D'
		file.Hash = *entry.IndexToWorkdir.OldFile.Oid

	} else if (entry.Status & git.StatusWtTypeChange) > 0 {
		file.Filename = entry.IndexToWorkdir.NewFile.Path
		file.Status.Unstaged = 'M'
		file.Hash = *entry.IndexToWorkdir.OldFile.Oid

	} else if (entry.Status & git.StatusWtRenamed) > 0 {
		file.Filename = entry.IndexToWorkdir.NewFile.Path
		file.Status.Unstaged = 'M'
		file.Hash = *entry.IndexToWorkdir.OldFile.Oid
	}

	// ----

	if (entry.Status & git.StatusConflicted) > 0 {
		file.Filename = entry.HeadToIndex.NewFile.Path
		file.Status.Unstaged = 'U'
		file.Hash = *entry.IndexToWorkdir.OldFile.Oid
	}
	// if (entry.Status & git.StatusIgnored) > 0 {
	// }

	stat, err := os.Stat(file.Filename)
	if err == nil {
		file.Size = uint64(stat.Size())
		file.Modified = uint32(stat.ModTime().Unix())
	}

	return file
}

// Returns the file list for a commit specified by its hash or a commit ref.
func (r *Repo) listFilesCommit(ctx context.Context, commitID CommitID) ([]File, error) {
	commit, err := r.ResolveCommit(commitID)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer commit.Free()

	tree, err := commit.Tree()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer tree.Free()

	files := []File{}
	err = tree.Walk(func(name string, entry *git.TreeEntry) int {
		select {
		case <-ctx.Done():
			return -1 // @@TODO: make sure this actually breaks the loop; docs aren't very clear
		default:
		}

		if entry.Filemode != git.FilemodeBlob && entry.Filemode != git.FilemodeBlobExecutable {
			return 0
		}

		blob, err := r.LookupBlob(entry.Id)
		if err != nil {
			log.Errorln("error looking up blob:", err)
			return 0
		}
		defer blob.Free()

		files = append(files, File{
			Filename: entry.Name,
			Hash:     *entry.Id,
			Status: Status{
				Unstaged: ' ',
				Staged:   ' ',
			},
			Size:     uint64(blob.Size()),
			Modified: 0,
		})

		return 0
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return files, nil
}

func (r *Repo) GetDiff(ctx context.Context, commitID CommitID) (io.Reader, error) {
	if commitID.Ref == "working" {
		return r.GetDiffWorktree(ctx)
	} else {
		return r.GetDiffCommit(ctx, commitID)
	}
}

func (r *Repo) GetDiffCommit(ctx context.Context, commitID CommitID) (io.Reader, error) {
	commit, err := r.ResolveCommit(commitID)
	if err != nil {
		return nil, err
	}
	defer commit.Free()

	// @@TODO: handle merges differently?
	if commit.ParentCount() > 1 {
		return nil, nil
	}

	commitTree, err := commit.Tree()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer commitTree.Free()

	var commitParentTree *git.Tree
	commitParent := commit.Parent(0)
	if commitParent != nil {
		defer commitParent.Free()

		commitParentTree, err = commitParent.Tree()
		if err != nil {
			return nil, errors.WithStack(err)
		}
		defer commitParentTree.Free()
	}

	diffOpts, err := git.DefaultDiffOptions()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	diff, err := r.DiffTreeToTree(commitParentTree, commitTree, &diffOpts)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return pipeDiff(diff), nil
}

func (r *Repo) GetDiffWorktree(ctx context.Context) (io.Reader, error) {
	var mostRecentCommitTree *git.Tree

	headRef, err := r.Head()
	if err != nil {
		// This is probably a new repository with no commits.  By passing a nil tree into
		// r.DiffTreeToWorkdir(...), we will receive a diff containing the full contents of the workdir.
	} else {
		defer headRef.Free()

		headRef, err = headRef.Resolve()
		if err != nil {
			return nil, err
		}
		defer headRef.Free()

		commitOid := headRef.Target()
		commit, err := r.LookupCommit(commitOid)
		if err != nil {
			return nil, err
		}
		defer commit.Free()

		mostRecentCommitTree, err = commit.Tree()
		if err != nil {
			return nil, err
		}
		defer mostRecentCommitTree.Free()
	}

	diffOpts, err := git.DefaultDiffOptions()
	if err != nil {
		return nil, err
	}

	diff, err := r.DiffTreeToWorkdir(mostRecentCommitTree, &diffOpts)
	if err != nil {
		return nil, err
	}

	return pipeDiff(diff), nil
}

func pipeDiff(diff *git.Diff) io.Reader {
	pr, pw := io.Pipe()

	go func() {
		var err error
		defer diff.Free()
		defer func() { pw.CloseWithError(err) }()

		numDeltas, err := diff.NumDeltas()
		if err != nil {
			err = errors.WithStack(err)
			return
		}

		for i := 0; i < numDeltas; i++ {
			patch, err := diff.Patch(i)
			if err != nil {
				err = errors.WithStack(err)
				return
			}

			patchStr, err := patch.String()
			if err != nil {
				err = errors.WithStack(err)
				return
			}

			_, err = pw.Write([]byte(patchStr))
			if err != nil {
				err = errors.WithStack(err)
				return
			}
		}
	}()

	return pr
}
