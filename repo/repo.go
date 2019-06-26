package repo

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/git-lfs/git-lfs/git/gitattr"
	"github.com/git-lfs/wildmatch"
	"github.com/libgit2/git2go"
	"github.com/pkg/errors"

	"github.com/Conscience/protocol/filters/encode"
	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/util"
)

type Repo struct {
	*git.Repository
	path string
}

const (
	CONSCIENCE_DATA_SUBDIR = "data"
	CONSCIENCE_HASH_LENGTH = 32
	GIT_HASH_LENGTH        = 20
)

var (
	Err404 = errors.New("not found")
)

type InitOptions struct {
	RepoID    string
	RepoRoot  string
	UserName  string
	UserEmail string
}

func Init(opts *InitOptions) (*Repo, error) {
	cRepo, err := git.InitRepository(opts.RepoRoot, false)
	if err != nil {
		return nil, errors.Wrapf(err, "could not initialize repo at path '%v'", opts.RepoRoot)
	}

	r := &Repo{Repository: cRepo, path: opts.RepoRoot}

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

func Open(repoRoot string) (*Repo, error) {
	gitRepo, err := git.OpenRepository(repoRoot)
	if err != nil && git.IsErrorCode(err, git.ErrNotFound) {
		return nil, errors.WithStack(Err404)
	} else if err != nil {
		return nil, errors.WithStack(err)
	}

	return &Repo{Repository: gitRepo, path: repoRoot}, nil
}

func (r *Repo) Path() string {
	// p := strings.Replace(r.Repository.Path(), "/.git/", "", -1)
	// return p
	return r.path
}

func (r *Repo) RepoID() (string, error) {
	cfg, err := r.Config()
	if err != nil {
		return "", errors.Wrapf(err, "could not open repo config at path '%v'", r.Path())
	}
	defer cfg.Free()

	repoID, err := cfg.LookupString("axon.repoid")
	if err != nil {
		return "", errors.Wrapf(err, "error looking up axon.repoid in .git/config (path: %v)", r.Path())
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

		odbObj, err := odb.Read(util.OidFromBytes(objectID))
		if err != nil {
			return nil, errors.WithStack(err)
		}

		if odbObj.Len() > 100*1024*1024 {
			repoID, _ := r.RepoID() // Intentionally ignoring error.  This is just to augment the existing error information.
			err := errors.Errorf("got request to open object over 100mb: %v %0x", repoID, objectID)
			log.Errorln(err)
			return nil, err
		}

		or := &objectReader{
			Reader: bytes.NewReader(odbObj.Data()),
			Closer: FuncCloser(func() error {
				// We have to manually .Free() the OdbObject (or, alternatively, call runtime.KeepAlive
				// on it) because otherwise the byte slice we obtain from .Data() will be deallocated
				// by the Go GC.
				odbObj.Free()
				return nil
			}),
			objectType: odbObj.Type(),
			objectLen:  odbObj.Len(),
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

	var entry *git.TreeEntry
	// Intentionally ignoring error because breaking the walk is an error
	_ = tree.Walk(func(fn string, innerEntry *git.TreeEntry) int {
		// log.Warnf("< %v >/( %v )", fn, entry.Name)
		if filepath.Join(fn, innerEntry.Name) == filename {
			entry = innerEntry
			return -1
		}
		return 0
	})
	if entry == nil {
		return nil, errors.WithStack(Err404)
	}

	// treeEntry := tree.EntryByName(filename)
	// if treeEntry == nil {
	// 	log.Warnf("OpenFileAtCommit treeEntry == nil")
	// 	return nil, errors.WithStack(Err404)
	// }

	return r.OpenObject(entry.Id[:])
}

func (r *Repo) ResolveCommitHash(commitID CommitID) (git.Oid, error) {
	if commitID.Ref != "" {
		// @@TODO: figure out if there are references that'll fail to resolve because we're using
		// Revparse instead of References.Lookup
		// ref, err := r.References.Lookup(commitID.Ref)
		// if err != nil {
		// 	return git.Oid{}, err
		// }
		// defer ref.Free()

		// ref, err = ref.Resolve()
		// if err != nil {
		// 	return git.Oid{}, err
		// }
		// defer ref.Free()

		// oid := ref.Target()
		// if oid == nil {
		// 	return git.Oid{}, errors.Errorf("could not resolve commit ref '%s' to a revision", commitID.Ref)
		// }
		// return *oid, nil

		obj, err := r.RevparseSingle(commitID.Ref)
		if err != nil {
			return git.Oid{}, err
		}
		return *obj.Id(), nil

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

	_repoID, err := cfg.LookupString("axon.repoid")
	if err != nil || _repoID != repoID {
		err = cfg.SetString("axon.repoid", repoID)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	cleanFilter, err := cfg.LookupString("filter.axon.clean")
	if err != nil || cleanFilter != "axon_encode" {
		err = cfg.SetString("filter.axon.clean", "axon_encode %f")
		if err != nil {
			return errors.WithStack(err)
		}
	}

	smudgeFilter, err := cfg.LookupString("filter.axon.smudge")
	if err != nil || smudgeFilter != "axon_encode" {
		err = cfg.SetString("filter.axon.smudge", "axon_decode %f")
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

			if url == "axon://"+repoID {
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

			remote, err = r.Remotes.CreateWithFetchspec(remoteName, "axon://"+repoID, "+refs/heads/*:refs/remotes/"+remoteName+"/*")
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

func (r *Repo) AxonRemote() (*git.Remote, error) {
	remoteNames, err := r.Remotes.List()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	for i := range remoteNames {
		remote, err := r.Remotes.Lookup(remoteNames[i])
		if err != nil {
			return nil, errors.WithStack(err)
		}

		url := remote.Url()
		if url[0:len("axon://")] == "axon://" {
			return remote, nil
		}
	}
	return nil, errors.Wrapf(Err404, "could not find axon:// remote")
}

func (r *Repo) UserIdentityFromConfig() (name string, email string, err error) {
	cfg, err := r.Config()
	if err != nil {
		return "", "", err
	}

	userName, err := cfg.LookupString("user.name")
	if err != nil {
		return "", "", err
	}

	userEmail, err := cfg.LookupString("user.email")
	if err != nil {
		return "", "", err
	}

	return userName, userEmail, nil
}

type CommitOptions struct {
	Pathspecs []string
	Message   string
}

func (r *Repo) CommitCurrentWorkdir(opts *CommitOptions) (*git.Oid, error) {
	userName, userEmail, err := r.UserIdentityFromConfig()
	if err != nil {
		return nil, err
	}

	var (
		author    = &git.Signature{Name: userName, Email: userEmail, When: time.Now()}
		committer = &git.Signature{Name: userName, Email: userEmail, When: time.Now()}
	)

	// find which files to chunk
	toAdd := make([]string, 0)
	toChunk := make([]string, 0)
	{
		statusList, err := r.StatusList(&git.StatusOptions{
			Pathspec: opts.Pathspecs,
			Show:     git.StatusShowWorkdirOnly,
			Flags:    git.StatusOptIncludeUntracked,
		})
		if err != nil {
			return nil, err
		}
		defer statusList.Free()

		statusCount, err := statusList.EntryCount()
		if err != nil {
			return nil, err
		}

		for i := 0; i < statusCount; i++ {
			status, err := statusList.ByIndex(i)
			if err != nil {
				return nil, err
			}

			code := status.Status
			if code != git.StatusWtNew &&
				code != git.StatusWtModified &&
				code != git.StatusWtRenamed &&
				code != git.StatusWtTypeChange {
				break
			}

			filename := status.IndexToWorkdir.NewFile.Path
			shouldChunk, err := r.FileIsChunked(filename, nil)
			if err != nil {
				return nil, err
			}
			if shouldChunk {
				toChunk = append(toChunk, filename)
			} else {
				toAdd = append(toAdd, filename)
			}
		}
	}

	idx, err := r.Index()
	if err != nil {
		return nil, err
	}

	err = idx.AddAll(toAdd, git.IndexAddDefault, nil)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(toChunk); i++ {
		filename := toChunk[i]
		reader, err := encode.EncodeFile(r.Path(), filename)
		if err != nil {
			return nil, err
		}

		chunked, err := ioutil.ReadAll(reader)
		if err != nil {
			return nil, err
		}

		oid, err := r.CreateBlobFromBuffer(chunked)
		if err != nil {
			return nil, err
		}

		p := filepath.Join(r.Path(), filename)
		stat, err := os.Stat(p)
		if err != nil {
			return nil, err
		}

		entry := &git.IndexEntry{
			Ctime: git.IndexTime{
				Seconds:     int32(time.Now().Unix()),
				Nanoseconds: uint32(time.Now().UnixNano()),
			},
			Mtime: git.IndexTime{
				Seconds: int32(stat.ModTime().Unix()),
			},
			Mode: git.FilemodeBlob,
			Uid:  uint32(os.Getuid()),
			Gid:  uint32(os.Getgid()),
			Size: uint32(stat.Size()),
			Id:   oid,
			Path: filename,
		}

		err = idx.Add(entry)
		if err != nil {
			return nil, err
		}
	}

	err = idx.Write()
	if err != nil {
		return nil, err
	}

	treeOid, err := idx.WriteTree()
	if err != nil {
		return nil, err
	}

	tree, err := r.LookupTree(treeOid)
	if err != nil {
		return nil, err
	}

	headRef, err := r.Head()
	if git.IsErrorCode(err, git.ErrUnbornBranch) {
		return r.CreateCommit("HEAD", author, committer, opts.Message, tree)
	} else if err != nil {
		return nil, err
	}

	headRef, err = headRef.Resolve()
	if err != nil {
		return nil, err
	}

	headCommit, err := r.LookupCommit(headRef.Target())
	if err != nil {
		return nil, err
	}

	return r.CreateCommit("HEAD", author, committer, opts.Message, tree, headCommit)
}

type PackfileWriter interface {
	io.Writer
	Commit() error
	Free()
}

func (r *Repo) PackfileWriter() (PackfileWriter, error) {
	odb, err := r.Odb()
	if err != nil {
		return nil, err
	}

	// return git.NewIndexer(filepath.Join(r.Path(), ".git", "objects", "pack"), odb, func(stats git.TransferProgress) git.ErrorCode {
	// 	return git.ErrOk
	// })
	return odb.NewWritePack(func(stats git.TransferProgress) git.ErrorCode {
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

		files[i] = mapStatusEntry(entry, r.Path())

		attr, err := r.GetAttribute(git.AttributeCheckNoSystem, files[i].Filename, "filter")
		if err != nil {
			return nil, err
		}

		isChunked := string(attr) == "axon"
		files[i].IsChunked = isChunked
		// git2go does not know about our custom chunking filters
		if isChunked && files[i].Status.Unstaged == 'M' {
			isModified, err := r.hasChunkedFileBeenModified(entry)
			if err != nil {
				return nil, err
			}
			if !isModified {
				files[i].Status.Unstaged = ' '
			}
		}
	}

	// Add empty directories to the list (git doesn't recognize them)
	emptyDirs, err := getEmptyDirs(r.path)
	if err != nil {
		return nil, err
	}

	files = append(files, emptyDirs...)

	return files, nil
}

func getEmptyDirs(root string) ([]File, error) {
	stack := []string{"."}

	var emptyDirs []File

	for len(stack) > 0 {
		path := stack[0]
		fullpath := filepath.Join(root, stack[0])
		stack = stack[1:]

		files, err := ioutil.ReadDir(fullpath)
		if err != nil {
			return nil, err
		}

		if len(files) == 0 {
			// Stat the folder to get its modtime
			stat, err := os.Stat(fullpath)
			if err != nil {
				return nil, err
			}

			emptyDirs = append(emptyDirs, File{
				Filename: path + string(filepath.Separator) + ".",
				Modified: uint32(stat.ModTime().Unix()),
				Status: Status{
					Staged:   ' ',
					Unstaged: 'M',
				},
			})
		} else {
			for _, file := range files {
				if file.IsDir() && file.Name() != ".git" {
					stack = append(stack, filepath.Join(path, file.Name()))
				}
			}
		}
	}
	return emptyDirs, nil
}

// Simplifies the interpretation of 'status' for a UI that primarily needs to display information
// about files in the worktree
func mapStatusEntry(entry git.StatusEntry, repoRoot string) File {
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

	p := filepath.Join(repoRoot, file.Filename)
	stat, err := os.Stat(p)
	if err == nil {
		file.Size = uint64(stat.Size())
		file.Modified = uint32(stat.ModTime().Unix())
	}

	return file
}

func (r *Repo) hasChunkedFileBeenModified(entry git.StatusEntry) (bool, error) {
	odb, err := r.Odb()
	if err != nil {
		return false, err
	}

	oldOid := entry.IndexToWorkdir.OldFile.Oid
	oldObj, err := odb.Read(oldOid)
	if err != nil {
		return true, nil
	}

	path := entry.IndexToWorkdir.NewFile.Path
	encReader, err := encode.EncodeFile(r.Path(), path)
	if err != nil {
		return false, err
	}

	oldData := oldObj.Data()
	newData, err := ioutil.ReadAll(encReader)
	if err != nil {
		return false, err
	}

	return bytes.Compare(oldData, newData) == 0, nil
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

	odb, err := r.Odb()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer odb.Free()

	files := []File{}
	err = tree.Walk(func(relPath string, entry *git.TreeEntry) int {
		select {
		case <-ctx.Done():
			return -1 // @@TODO: make sure this actually breaks the loop; docs aren't very clear
		default:
		}

		if entry.Filemode != git.FilemodeBlob && entry.Filemode != git.FilemodeBlobExecutable {
			return 0
		}

		filename := filepath.Join(relPath, entry.Name)

		// Grab the file's filter attribute (if any) so that we can determine if it's chunked or not
		filterAttrValue, _, _, _, err := r.getAttributeFileWithAttributeInTree(filename, "filter", tree)
		if err != nil {
			log.Errorln("error looking up file's filter attribute:", err)
		}

		// Fetch the file's size
		size, _, err := odb.ReadHeader(entry.Id)
		if err != nil {
			log.Errorln("error looking up file's size:", err)
		}

		modified := uint32(commit.Author().When.Unix())

		files = append(files, File{
			Filename: filename,
			Hash:     *entry.Id,
			Status: Status{
				Unstaged: ' ',
				Staged:   ' ',
			},
			Size:      size,
			Modified:  modified,
			IsChunked: string(filterAttrValue) == "axon",
		})

		return 0
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return files, nil
}

// The returned io.Reader is nil when a diff for a merge commit is fetched.
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
				patch.Free()
				err = errors.WithStack(err)
				return
			}
			patch.Free()

			_, err = pw.Write([]byte(patchStr))
			if err != nil {
				err = errors.WithStack(err)
				return
			}
		}
	}()

	return pr
}

func (r *Repo) FileIsChunked(filename string, commitID *git.Oid) (bool, error) {
	// for current worktree
	if commitID == nil {
		filterName, _, _, _, err := r.getAttributeFileWithAttribute(filename, "filter")
		if err != nil {
			return false, err
		}
		return filterName == "axon", nil
	}
	commit, err := r.LookupCommit(commitID)
	if err != nil {
		return false, errors.WithStack(err)
	}

	tree, err := commit.Tree()
	if err != nil {
		return false, errors.WithStack(err)
	}

	filterName, _, _, _, err := r.getAttributeFileWithAttributeInTree(filename, "filter", tree)
	if err != nil {
		return false, errors.WithStack(err)
	}

	return filterName == "axon", nil
}

func (r *Repo) SetFileChunking(filename string, shouldEnable bool) error {
	attrValue, lineIndex, attrIndex, attrFile, err := r.getAttributeFileWithAttribute(filename, "filter")
	if err != nil {
		return errors.WithStack(err)
	}

	isEnabled := string(attrValue) == "axon"

	if shouldEnable && !isEnabled {
		// create a .gitattributes file at the root if it doesn't exist
		if attrFile == nil {
			attrFile = &AttrFile{Path: filepath.Join(r.Path(), ".gitattributes")}
		}

		if lineIndex > -1 {
			if attrIndex > -1 {
				// change the existing attribute on the existing line
				attrFile.lines[lineIndex].Attrs[attrIndex].V = "axon"
			} else {
				// add a new attribute to the existing line
				attrFile.lines[lineIndex].Attrs = append(attrFile.lines[lineIndex].Attrs, &gitattr.Attr{K: "filter", V: "axon"})
			}
		} else {
			// append a new line referring to this file
			attrFile.lines = append(attrFile.lines, createAttrLine(filename, "filter", "axon"))
		}

		// write the .gitattributes file
		return attrFile.write()

	} else if !shouldEnable && isEnabled {
		// remove the attribute
		attrFile.lines[lineIndex].Attrs = append(attrFile.lines[lineIndex].Attrs[:attrIndex], attrFile.lines[lineIndex].Attrs[attrIndex+1:]...)

		// write the .gitattributes file
		return attrFile.write()
	}

	return nil
}

type AttrFile struct {
	Path  string
	lines []*gitattr.Line
}

func (af *AttrFile) write() error {
	f, err := os.Create(af.Path)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, line := range af.lines {
		if len(line.Attrs) == 0 {
			// skip lines with no attributes
			continue
		}

		var attrs []string
		for _, attr := range line.Attrs {
			if attr.Unspecified {
				attrs = append(attrs, "!"+attr.K)
			} else {
				attrs = append(attrs, attr.K+"="+attr.V)
			}
		}
		f.WriteString(fmt.Sprintf("%s %s\n", line.Pattern, strings.Join(attrs, " ")))
	}
	return nil
}

func createAttrLine(pattern, key, value string) *gitattr.Line {
	return &gitattr.Line{
		Pattern: wildmatch.NewWildmatch(pattern),
		Attrs: []*gitattr.Attr{
			{K: key, V: value},
		},
	}
}

func (r *Repo) getAttributeFileWithAttribute(filename, attrName string) (attrValue string, lineIndex int, attrIndex int, attrFile *AttrFile, err error) {
	dir, _ := filepath.Split(filename)
	pathParts := strings.Split(dir, string(filepath.Separator))
	pathParts = append([]string{"."}, pathParts[:len(pathParts)-1]...)

	repoRoot := r.Path()
	for i := len(pathParts); i > 0; i-- {
		currentDir := filepath.Join(pathParts[:i]...)
		attrFilePath := filepath.Join(currentDir, ".gitattributes")

		lines, err := parseGitAttributes(repoRoot, attrFilePath)
		if err != nil {
			continue
		}

		for lineIndex, line := range lines {
			relPath, err := filepath.Rel(currentDir, filename)
			if err != nil {
				log.Errorln(err)
				continue
			}

			if line.Pattern != nil && line.Pattern.Match(relPath) {
				for attrIndex, attr := range line.Attrs {
					if attr.K == attrName {
						return attr.V, lineIndex, attrIndex, &AttrFile{Path: filepath.Join(r.Path(), attrFilePath), lines: lines}, nil
					}
				}
			}
		}
	}
	// if attr isn't in a file, return root gitattributes
	attrFilePath := filepath.Join(repoRoot, ".gitattributes")
	lines, err := parseGitAttributes(repoRoot, ".gitattributes")
	if err != nil {
		lines = []*gitattr.Line{}
	}
	return "", -1, -1, &AttrFile{Path: attrFilePath, lines: lines}, nil
}

func parseGitAttributes(repoRoot, path string) ([]*gitattr.Line, error) {
	f, err := os.Open(filepath.Join(repoRoot, path))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	x, _, err := gitattr.ParseLines(f)
	return x, err
}

func (r *Repo) getAttributeFileWithAttributeInTree(filename, attrName string, tree *git.Tree) (attrValue string, lineIndex int, attrIndex int, attrFile *AttrFile, err error) {
	dir, _ := filepath.Split(filename)
	pathParts := strings.Split(dir, string(filepath.Separator))
	pathParts = append([]string{"."}, pathParts[:len(pathParts)-1]...)

	odb, err := r.Odb()
	if err != nil {
		return "", 0, 0, nil, err
	}

	repoRoot := r.Path()
	for i := len(pathParts); i > 0; i-- {
		currentDir := filepath.Join(pathParts[:i]...)
		attrFilePath := filepath.Join(currentDir, ".gitattributes")

		attrEntry, err := tree.EntryByPath(attrFilePath)
		if err != nil {
			continue
		}

		obj, err := odb.Read(attrEntry.Id)
		if err != nil {
			continue
		}
		defer obj.Free()

		lines, _, err := gitattr.ParseLines(bytes.NewReader(obj.Data()))
		if err != nil {
			continue
		}

		for lineIndex, line := range lines {
			relPath, err := filepath.Rel(currentDir, filename)
			if err != nil {
				continue
			}

			if line.Pattern != nil && line.Pattern.Match(relPath) {
				for attrIndex, attr := range line.Attrs {
					if attr.K == attrName {
						return attr.V, lineIndex, attrIndex, &AttrFile{Path: filepath.Join(repoRoot, attrFilePath), lines: lines}, nil
					}
				}
			}
		}
	}
	return "", 0, 0, nil, nil
}

func (r *Repo) FilesChangedByCommit(ctx context.Context, commitID *git.Oid) ([]File, error) {
	newCommit, err := r.LookupCommit(commitID)
	if err != nil {
		return nil, err
	}

	if newCommit.ParentCount() == 0 {
		return r.filesChangedByCommit_noParent(ctx, newCommit)
	} else {
		return r.filesChangedByCommit_hasParent(ctx, newCommit)
	}
}

func (r *Repo) filesChangedByCommit_hasParent(ctx context.Context, newCommit *git.Commit) ([]File, error) {
	oldCommit := newCommit.Parent(0)

	newTree, err := newCommit.Tree()
	if err != nil {
		return nil, err
	}

	oldTree, err := oldCommit.Tree()
	if err != nil {
		return nil, err
	}

	diffOpts, err := git.DefaultDiffOptions()
	if err != nil {
		return nil, err
	}

	// @@TODO: figure out the cheapest diff we can run and still get filenames back.  This is my
	// first attempt at that.
	diffOpts.Flags |= git.DiffMinimal

	diff, err := r.DiffTreeToTree(oldTree, newTree, &diffOpts)
	if err != nil {
		return nil, err
	}

	var files []File

	callback := func(delta git.DiffDelta, x float64) (git.DiffForEachHunkCallback, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		files = append(files, File{Filename: delta.NewFile.Path})
		return nil, nil
	}

	err = diff.ForEach(callback, git.DiffDetailFiles)
	if err != nil {
		return nil, err
	}

	return files, nil
}

func (r *Repo) filesChangedByCommit_noParent(ctx context.Context, commit *git.Commit) ([]File, error) {
	return r.listFilesCommit(ctx, CommitID{Hash: commit.Id()})
}
