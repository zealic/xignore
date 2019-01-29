package xignore

import (
	"os"
	"sort"

	"github.com/spf13/afero"
)

// MatchesOptions matches options
type MatchesOptions struct {
	// Ignorefile name, similar '.gitignore', '.dockerignore', 'chefignore'
	Ignorefile string
	// Allow nested ignorefile
	Nested bool
}

// MatchesResult matches result
type MatchesResult struct {
	BaseDir string
	// ignorefile rules matched files
	MatchedFiles []string
	// ignorefile rules unmatched files
	UnmatchedFiles []string
	// ignorefile rules matched dirs
	MatchedDirs []string
	// ignorefile rules unmatched dirs
	UnmatchedDirs []string
	// error files when return error
	ErrorFiles []string
	// error files when return error
	ErrorDirs []string
}

// Matcher xignore matcher
type Matcher struct {
	fs afero.Fs
}

// NewSystemMatcher create matcher for system filesystem
func NewSystemMatcher() *Matcher {
	return &Matcher{afero.NewReadOnlyFs(afero.NewOsFs())}
}

// Matches returns matched files from dir files.
func (m *Matcher) Matches(basedir string, options *MatchesOptions) (*MatchesResult, error) {
	vfs := afero.NewBasePathFs(m.fs, basedir)
	ignorefile := options.Ignorefile
	if ok, err := afero.DirExists(vfs, "/"); !ok || err != nil {
		if err == nil {
			return nil, os.ErrNotExist
		}
		return nil, err
	}

	// Root filemap
	fileMap, errorFiles, err := createFileStateMap(vfs, ignorefile, true)
	if err != nil {
		return nil, err
	}

	// Apply nested filemap
	if options.Nested {
		nestedErrorFile, err := fileMap.applyIgnorefile(vfs, ignorefile)
		if err != nil {
			return nil, err
		}
		for _, efile := range nestedErrorFile {
			errorFiles = append(errorFiles, efile)
		}
	}

	return makeResult(vfs, basedir, fileMap, errorFiles)
}

func makeResult(vfs afero.Fs, basedir string,
	fileMap stateMap, errorFiles []string) (*MatchesResult, error) {
	matchedFiles := []string{}
	unmatchedFiles := []string{}
	matchedDirs := []string{}
	unmatchedDirs := []string{}
	errorDirs := []string{}
	for f, matched := range fileMap {
		if f == "" {
			continue
		}
		isDir, err := afero.IsDir(vfs, f)
		if err != nil {
			errorDirs = append(errorDirs, f)
			return nil, err
		}
		if isDir {
			if matched {
				matchedDirs = append(matchedDirs, f)
			} else {
				unmatchedDirs = append(unmatchedDirs, f)
			}
		} else {
			if matched {
				matchedFiles = append(matchedFiles, f)
			} else {
				unmatchedFiles = append(unmatchedFiles, f)
			}
		}
	}

	sort.Strings(matchedFiles)
	sort.Strings(unmatchedFiles)
	sort.Strings(errorFiles)
	sort.Strings(matchedDirs)
	sort.Strings(unmatchedDirs)
	sort.Strings(errorDirs)
	return &MatchesResult{
		BaseDir:        basedir,
		MatchedFiles:   matchedFiles,
		UnmatchedFiles: unmatchedFiles,
		ErrorFiles:     errorFiles,
		MatchedDirs:    matchedDirs,
		UnmatchedDirs:  unmatchedDirs,
		ErrorDirs:      errorDirs,
	}, nil
}
