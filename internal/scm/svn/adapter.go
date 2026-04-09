package svn

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"gig/internal/scm"
)

type Adapter struct{}

func NewAdapter() *Adapter {
	return &Adapter{}
}

func (a *Adapter) Type() scm.Type {
	return scm.TypeSVN
}

func (a *Adapter) DetectRoot(path string) (string, bool, error) {
	start, err := normalizePath(path)
	if err != nil {
		return "", false, err
	}

	for {
		ok, err := a.IsRepository(start)
		if err != nil {
			return "", false, err
		}
		if ok {
			return start, true, nil
		}

		parent := filepath.Dir(start)
		if parent == start {
			return "", false, nil
		}
		start = parent
	}
}

func (a *Adapter) IsRepository(path string) (bool, error) {
	_, err := os.Stat(filepath.Join(path, ".svn"))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	return false, err
}

func (a *Adapter) CurrentBranch(context.Context, string) (string, error) {
	return "", scm.ErrUnsupported
}

func (a *Adapter) SearchCommits(context.Context, string, scm.SearchQuery) ([]scm.Commit, error) {
	return nil, scm.ErrUnsupported
}

func (a *Adapter) CompareBranches(context.Context, string, scm.CompareQuery) (scm.CompareResult, error) {
	return scm.CompareResult{}, scm.ErrUnsupported
}

func (a *Adapter) PrepareCherryPick(context.Context, string, scm.CherryPickPlan) error {
	return scm.ErrUnsupported
}

func normalizePath(path string) (string, error) {
	if path == "" {
		path = "."
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", err
	}

	if info.IsDir() {
		return absPath, nil
	}

	return filepath.Dir(absPath), nil
}
