package xignore

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

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
	BaseDir        string
	MatchedFiles   []string
	UnmatchedFiles []string
	ErrorFiles     []string
	MatchedDirs    []string
	UnmatchedDirs  []string
	ErrorDirs      []string
}

// Matcher xignore matcher
type Matcher struct {
	fs afero.Fs
}

// NewSystemMatcher create matcher for system filesystem
func NewSystemMatcher() *Matcher {
	return &Matcher{afero.NewReadOnlyFs(afero.NewOsFs())}
}

func collectFiles(fs afero.Fs) (files []string, errFiles []string) {
	files = []string{}
	errFiles = []string{}

	afero.Walk(fs, "", func(path string, info os.FileInfo, werr error) error {
		if werr != nil {
			errFiles = append(errFiles, path)
		} else {
			files = append(files, path)
		}
		return nil
	})
	return
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
	fileMap, errorFiles, err := getFileMap(vfs, ignorefile, true)
	if err != nil {
		return nil, err
	}

	// Apply nested filemap
	if options.Nested {
		nestedErrorFile, err := applyNestedFileMap(vfs, ignorefile, fileMap)
		if err != nil {
			return nil, err
		}
		for _, efile := range nestedErrorFile {
			if ok := fileMap[efile]; ok {
				delete(fileMap, efile)
			}
		}
	}

	return makeResult(vfs, basedir, fileMap, errorFiles), nil
}

func applyNestedFileMap(vfs afero.Fs, ignorefile string, fileMap map[string]bool) ([]string, error) {
	// Apply nested ignorefile
	nestedIgnorefiles := []string{}
	for file := range fileMap {
		// all subdir ignorefiles
		if strings.HasSuffix(file, ignorefile) && len(file) != len(ignorefile) {
			nestedIgnorefiles = append(nestedIgnorefiles, file)
		}
	}
	// Sort by dir deep level
	sort.Slice(nestedIgnorefiles, func(i, j int) bool {
		return len(filepath.SplitList(nestedIgnorefiles[i])) < len(filepath.SplitList(nestedIgnorefiles[j]))
	})

	errorFiles := []string{}
	for _, ifile := range nestedIgnorefiles {
		nestedBasedir := filepath.Dir(ifile)
		nestedFs := afero.NewBasePathFs(vfs, nestedBasedir)
		nestedFileMap, errorFiles, err := getFileMap(nestedFs, ignorefile, false)
		if err != nil {
			return nil, err
		}
		for _, efile := range errorFiles {
			errorFiles = append(errorFiles, filepath.Join(nestedBasedir, efile))
		}

		for nfile, matched := range nestedFileMap {
			parentFile := filepath.Join(nestedBasedir, nfile)
			fileMap[parentFile] = matched
		}
	}

	return errorFiles, nil
}

func getPatterns(vfs afero.Fs, ignorefile string) ([]*Pattern, error) {
	// read ignorefile
	ignoreFilePath := ignorefile
	if ignoreFilePath == "" {
		ignoreFilePath = DefaultIgnorefile
	}
	exists, err := afero.Exists(vfs, ignoreFilePath)
	if err != nil {
		return nil, err
	}

	// Load patterns from ignorefile
	patterns := []*Pattern{}
	if exists {
		f, err := vfs.Open(ignoreFilePath)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		ignoreFile := Ignorefile{}
		err = ignoreFile.FromReader(f)
		if err != nil {
			return nil, err
		}
		for _, sp := range ignoreFile.Patterns {
			patterns = append(patterns, NewPattern(sp))
		}
	}

	return patterns, nil
}

func getFileMap(vfs afero.Fs, ignorefile string, rootMap bool) (map[string]bool, []string, error) {
	// Collect all files
	files, errorFiles := collectFiles(vfs)
	fileMap := map[string]bool{}
	if rootMap {
		for _, f := range files {
			fileMap[f] = false
		}
	}

	// matching patterns
	patterns, err := getPatterns(vfs, ignorefile)
	if err != nil {
		return nil, nil, err
	}
	for _, pattern := range patterns {
		if pattern.IsEmpty() {
			continue
		}
		currFiles, err := afero.Glob(vfs, pattern.value)
		if err != nil {
			return nil, nil, err
		}
		if pattern.IsExclusion() {
			for _, f := range currFiles {
				fileMap[f] = false
			}
		} else {
			for _, f := range currFiles {
				fileMap[f] = true
			}
		}
	}

	return fileMap, errorFiles, nil
}

func makeResult(vfs afero.Fs, basedir string,
	fileMap map[string]bool, errorFiles []string) *MatchesResult {
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
			continue
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
	}
}
