package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	gitconfig "gopkg.in/src-d/go-git.v4/plumbing/format/config"
)

func getRefs() ([]string, error) {
	refs, err := client.GetAllRefs(repoID)
	if err != nil {
		return nil, err
	}
	refsList := make([]string, 0)
	for name, target := range refs {
		refsList = append(refsList, fmt.Sprintf("%s %s", target, name))
	}
	return refsList, nil
}

func setupConfig() error {
	cfg, err := repo.Config()
	if err != nil {
		return err
	}

	raw := cfg.Raw
	changed := false
	section := raw.Section("conscience")
	if section.Option("username") != repoUser {
		raw.SetOption("conscience", "", "username", repoUser)
		changed = true
	}
	if section.Option("reponame") != repoName {
		raw.SetOption("conscience", "", "reponame", repoName)
		changed = true
	}

	filter := raw.Section("filter").Subsection("conscience")
	if filter.Option("clean") != "conscience_encode" {
		raw.SetOption("filter", "conscience", "clean", "conscience_encode")
		changed = true
	}
	if filter.Option("smudge") != "conscience_decode" {
		raw.SetOption("filter", "conscience", "smudge", "conscience_decode")
		changed = true
	}

	if changed {
		p := filepath.Join(GIT_DIR, "config")
		f, err := os.OpenFile(p, os.O_WRONLY, os.ModeAppend)
		if err != nil {
			return err
		}
		w := io.Writer(f)

		enc := gitconfig.NewEncoder(w)
		err = enc.Encode(raw)
		if err != nil {
			return err
		}
	}

	return nil
}
