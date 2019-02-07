package nodegit

import (
	"context"
	"path/filepath"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/swarm/noderpc/pb"
	"github.com/Conscience/protocol/util"
)

func ListFiles(ctx context.Context, path string, commit string) ([]*pb.File, error) {
	files := map[string]*pb.File{}

	// Start by taking the output of `git ls-files --stage`
	err := util.ExecAndScanStdout(ctx, []string{"git", "ls-files", "--stage"}, path, func(line string) error {
		file, err := parseGitLSFilesLine(line)
		if err != nil {
			log.Errorln("GetRepoFiles (git ls-files):", err)
			return nil // continue
		}
		files[file.Name] = file
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Then, overlay the output of `git status --porcelain`
	err = util.ExecAndScanStdout(ctx, []string{"git", "status", "--porcelain=2"}, path, func(line string) error {
		file, err := parseGitStatusLine(line)
		if err != nil {
			log.Errorln("GetRepoFiles (git status --porcelain=2):", err)
			return nil // continue
		}
		files[file.Name] = file
		return nil
	})
	if err != nil {
		return nil, err
	}

	unresolved, mergeConflicts, err := getMergeConflicts(ctx, path)
	if err != nil {
		return nil, err
	}

	fileList := []*pb.File{}
	for _, file := range files {
		stat, err := getStats(filepath.Join(path, file.Name))
		if err != nil {
			log.Errorln("GetRepoFiles (getStats):", err)
			continue
		}
		file.Modified = uint32(stat.ModTime().Unix())
		file.Size = uint64(stat.Size())
		file.MergeConflict = contains(mergeConflicts, file.Name)
		file.MergeUnresolved = contains(unresolved, file.Name)
		fileList = append(fileList, file)
	}

	return fileList, nil
}
