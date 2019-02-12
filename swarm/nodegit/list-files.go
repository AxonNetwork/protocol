package nodegit

import (
	"context"

	"github.com/Conscience/protocol/util"
)

// import (
// 	"context"
// 	"encoding/hex"
// 	"fmt"
// 	"os"
// 	"path/filepath"
// 	"strconv"
// 	"strings"

// 	"github.com/Conscience/protocol/log"
// 	"github.com/Conscience/protocol/swarm/noderpc/pb"
// 	"github.com/Conscience/protocol/util"
// )

func GetFilesForCommit(ctx context.Context, path string, commitHash string) ([]string, error) {
	files := make([]string, 0)
	err := util.ExecAndScanStdout(ctx, []string{"git", "show", "--name-only", "--pretty=format:\"\"", commitHash}, path, func(line string) error {
		if len(line) > 0 {
			files = append(files, line)
		}
		return nil
	})
	return files, err
}

// func ListFiles(ctx context.Context, path string, commit string) ([]*pb.File, error) {
// 	files := map[string]*pb.File{}

// 	// Start by taking the output of `git ls-files --stage`
// 	err := util.ExecAndScanStdout(ctx, []string{"git", "ls-files", "--stage"}, path, func(line string) error {
// 		file, err := parseGitLSFilesLine(line)
// 		if err != nil {
// 			log.Errorln("GetRepoFiles (git ls-files):", err)
// 			return nil // continue
// 		}
// 		files[file.Name] = file
// 		return nil
// 	})
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Then, overlay the output of `git status --porcelain`
// 	err = util.ExecAndScanStdout(ctx, []string{"git", "status", "--porcelain=2"}, path, func(line string) error {
// 		file, err := parseGitStatusLine(line)
// 		if err != nil {
// 			log.Errorln("GetRepoFiles (git status --porcelain=2):", err)
// 			return nil // continue
// 		}
// 		files[file.Name] = file
// 		return nil
// 	})
// 	if err != nil {
// 		return nil, err
// 	}

// 	unresolved, mergeConflicts, err := getMergeConflicts(ctx, path)
// 	if err != nil {
// 		return nil, err
// 	}

// 	fileList := []*pb.File{}
// 	for _, file := range files {
// 		stat, err := getStats(filepath.Join(path, file.Name))
// 		if err != nil {
// 			log.Errorln("GetRepoFiles (getStats):", err)
// 			continue
// 		}
// 		file.Modified = uint32(stat.ModTime().Unix())
// 		file.Size = uint64(stat.Size())
// 		file.MergeConflict = contains(mergeConflicts, file.Name)
// 		file.MergeUnresolved = contains(unresolved, file.Name)
// 		fileList = append(fileList, file)
// 	}

// 	return fileList, nil
// }

// func parseGitStatusLine(line string) (*pb.File, error) {
// 	parts := strings.Split(line, " ")
// 	file := &pb.File{}

// 	switch parts[0] {
// 	case "u":
// 		mode, err := strconv.ParseUint(parts[3], 8, 32)
// 		if err != nil {
// 			return nil, err
// 		}

// 		hash, err := hex.DecodeString(parts[7])
// 		if err != nil {
// 			return nil, err
// 		}

// 		file.Name = parts[10]
// 		file.Hash = hash
// 		file.Mode = uint32(mode)
// 		file.UnstagedStatus = parts[1][:1]
// 		file.StagedStatus = parts[1][1:]

// 	case "1":
// 		mode, err := strconv.ParseUint(parts[3], 8, 32)
// 		if err != nil {
// 			return nil, err
// 		}

// 		hash, err := hex.DecodeString(parts[6])
// 		if err != nil {
// 			return nil, err
// 		}

// 		file.Name = strings.Join(parts[8:], " ")
// 		file.Hash = hash
// 		file.Mode = uint32(mode)
// 		file.StagedStatus = parts[1][:1]
// 		file.UnstagedStatus = parts[1][1:]

// 	case "2":
// 		// @@TODO: these are renames

// 	case "?":
// 		file.Name = strings.Join(parts[1:], " ")
// 		file.UnstagedStatus = "?"
// 		file.StagedStatus = "?"
// 	}

// 	return file, nil
// }

// func parseGitLSFilesLine(line string) (*pb.File, error) {
// 	moarParts := strings.Split(line, "\t")
// 	parts := strings.Split(moarParts[0], " ")

// 	mode, err := strconv.ParseUint(parts[0], 8, 32)
// 	if err != nil {
// 		return nil, err
// 	}

// 	hash, err := hex.DecodeString(parts[1])
// 	if err != nil {
// 		return nil, err
// 	}

// 	name := moarParts[1]
// 	if name[0:1] == "\"" {
// 		name = fmt.Sprintf(name[1 : len(name)-2])
// 	}

// 	return &pb.File{
// 		Name:           name,
// 		Hash:           hash,
// 		Mode:           uint32(mode),
// 		Size:           0,
// 		UnstagedStatus: ".",
// 		StagedStatus:   ".",
// 	}, nil
// }

// func getStats(path string) (os.FileInfo, error) {
// 	f, err := os.Open(path)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer f.Close()
// 	stat, err := f.Stat()
// 	return stat, err
// }

// func getMergeConflicts(ctx context.Context, path string) ([]string, []string, error) {
// 	unresolved := make([]string, 0)
// 	err := util.ExecAndScanStdout(ctx, []string{"git", "diff", "--name-only", "--diff-filter=U"}, path, func(line string) error {
// 		unresolved = append(unresolved, line)
// 		return nil
// 	})
// 	if err != nil {
// 		return []string{}, []string{}, err
// 	}

// 	mergeConflicts := make([]string, 0)
// 	for i := range unresolved {
// 		exists, err := util.GrepExists(filepath.Join(path, unresolved[i]), "<<<<<")
// 		if err != nil {
// 			return []string{}, []string{}, err
// 		}
// 		if exists {
// 			mergeConflicts = append(mergeConflicts, unresolved[i])
// 		}

// 	}
// 	return unresolved, mergeConflicts, err
// }

// func contains(arr []string, str string) bool {
// 	for i := range arr {
// 		if arr[i] == str {
// 			return true
// 		}
// 	}
// 	return false
// }
