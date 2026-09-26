package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const sourceAddress = "https://github.com/cangrejometralleta/OneTwoThree.git"

type sourceTree struct {
	root     string
	commit   string
	dirty    bool
	files    map[string]bool
	excluded map[string]bool
}

func runGit(root string, args ...string) (string, error) {
	command := exec.Command("git", append([]string{"-C", root}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %s: %w", args[0], output, err)
	}
	return strings.TrimSuffix(string(output), "\n"), nil
}

func readSourceTree(options exportOptions) (sourceTree, error) {
	root, err := filepath.Abs(options.source)
	if err != nil {
		return sourceTree{}, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return sourceTree{}, err
	}
	top, err := runGit(root, "rev-parse", "--show-toplevel")
	if err != nil {
		return sourceTree{}, err
	}
	if filepath.Clean(top) != root {
		return sourceTree{}, fmt.Errorf("source must be the repository root")
	}
	origin, err := runGit(root, "remote", "get-url", "origin")
	if err != nil {
		return sourceTree{}, err
	}
	if origin != sourceAddress && origin != "git@github.com:cangrejometralleta/OneTwoThree.git" {
		return sourceTree{}, fmt.Errorf("source origin is not OneTwoThree: %s", origin)
	}
	commit, err := runGit(root, "rev-parse", "HEAD")
	if err != nil {
		return sourceTree{}, err
	}
	state, err := runGit(root, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return sourceTree{}, err
	}
	if state != "" && !options.allowDirty {
		return sourceTree{}, fmt.Errorf("source has local changes; use --allow-dirty for a labelled preview")
	}

	files, err := runGit(root, "ls-files", "--cached", "--others", "--exclude-standard", "-z")
	if err != nil {
		return sourceTree{}, err
	}
	excluded, err := runGit(root, "ls-files", "--cached", "--others", "--ignored", "--exclude-from=.canonignore", "-z")
	if err != nil {
		return sourceTree{}, err
	}
	return sourceTree{root, commit, state != "", splitPaths(files), splitPaths(excluded)}, nil
}

func splitPaths(text string) map[string]bool {
	paths := make(map[string]bool)
	for _, name := range strings.Split(text, "\x00") {
		if name != "" {
			paths[name] = true
		}
	}
	return paths
}

func (source sourceTree) readFile(name string) ([]byte, error) {
	if !source.files[name] || source.excluded[name] {
		return nil, fmt.Errorf("unavailable source: %s", name)
	}
	path := filepath.Join(source.root, filepath.FromSlash(name))
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, err
	}
	if resolved != path {
		return nil, fmt.Errorf("symbolic source is not supported: %s", name)
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file: %s", name)
	}
	return os.ReadFile(path)
}
