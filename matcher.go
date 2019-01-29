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
		nestedErrorFile, err := fileMap.applyIgnorefile(vfs, ignorefile)
		if err != nil {
			return nil, err
		}
		for _, efile := range nestedErrorFile {
			errorFiles = append(errorFiles, efile)
		}
	}

	return makeResult(vfs, basedir, fileMap, errorFiles), nil
}

func getPatterns(vfs afero.Fs, ignorefile string) ([]*Pattern, error) {
	// read ignorefile
	ignoreFilePath := ignorefile
	if ignoreFilePath == "" {
		ignoreFilePath = DefaultIgnorefile
	}
	ignoreExists, err := afero.Exists(vfs, ignoreFilePath)
	if err != nil {
		return nil, err
	}

	// Load patterns from ignorefile
	patterns := []*Pattern{}
	if ignoreExists {
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

func getFileMap(vfs afero.Fs, ignorefile string, rootMap bool) (fileMap, []string, error) {
	// Collect all files
	files, errorFiles := collectFiles(vfs)
	mainMap := fileMap{}
	if rootMap {
		for _, f := range files {
			mainMap[f] = false
		}
	}

	// matching patterns
	patterns, err := getPatterns(vfs, ignorefile)
	if err != nil {
		return nil, nil, err
	}
	// Prepare regexp
	for _, pattern := range patterns {
		err := pattern.Prepare()
		if err != nil {
			return nil, nil, err
		}
	}

	// files match
	filesMap := fileMap{}
	dirPatterns := []*Pattern{}
	for _, pattern := range patterns {
		if pattern.IsEmpty() {
			continue
		}
		currFiles := pattern.Matches(files)
		if pattern.IsExclusion() {
			for _, f := range currFiles {
				filesMap[f] = false
			}
		} else {
			for _, f := range currFiles {
				filesMap[f] = true
			}
		}

		// store matched/unmatched dirs
		for _, f := range currFiles {
			ok, err := afero.IsDir(vfs, f)
			if err != nil {
				return nil, nil, err
			}
			if ok {
				strPattern := f + "/**"
				if pattern.IsExclusion() {
					strPattern = "!" + strPattern
				}
				dirPattern := NewPattern(strPattern)
				dirPatterns = append(dirPatterns, dirPattern)
				err := dirPattern.Prepare()
				if err != nil {
					return nil, nil, err
				}
			}
		}
	}

	// handle dirs batch matches
	dirFileMap := map[string]bool{}
	for _, pattern := range dirPatterns {
		if pattern.IsEmpty() {
			continue
		}
		currFiles := pattern.Matches(files)
		if pattern.IsExclusion() {
			for _, f := range currFiles {
				dirFileMap[f] = false
			}
		} else {
			for _, f := range currFiles {
				dirFileMap[f] = true
			}
		}
	}

	// merge target
	mainMap.merge(dirFileMap)
	mainMap.merge(filesMap)

	return mainMap, errorFiles, nil
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

type fileMap map[string]bool

func (fileMap fileMap) merge(source fileMap) {
	for file, val := range source {
		fileMap[file] = val
	}
}

func (fileMap fileMap) applyIgnorefile(vfs afero.Fs, ignorefile string) ([]string, error) {
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
		ilen := len(strings.Split(nestedIgnorefiles[i], string(os.PathSeparator)))
		jlen := len(strings.Split(nestedIgnorefiles[j], string(os.PathSeparator)))
		return ilen < jlen
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

	// Remove error files
	for _, efile := range errorFiles {
		errorFiles = append(errorFiles, efile)
		if ok := fileMap[efile]; ok {
			delete(fileMap, efile)
		}
	}

	return errorFiles, nil
}
