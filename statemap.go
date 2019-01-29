package xignore

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/afero"
)

type stateMap map[string]bool

func createFileStateMap(vfs afero.Fs, ignorefile string, rootMap bool) (stateMap, []string, error) {
	// Collect all files
	files, errorFiles := collectFiles(vfs)
	mainMap := stateMap{}
	if rootMap {
		for _, f := range files {
			mainMap[f] = false
		}
	}

	// matching patterns
	patterns, err := loadPatterns(vfs, ignorefile)
	if err != nil {
		return nil, nil, err
	}

	// files match
	filesMap := stateMap{}
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
	dirFileMap := stateMap{}
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

func loadPatterns(vfs afero.Fs, ignorefile string) ([]*Pattern, error) {
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
			pattern := NewPattern(sp)
			err := pattern.Prepare()
			if err != nil {
				return nil, err
			}
			patterns = append(patterns, pattern)
		}
	}

	return patterns, nil
}

func (state stateMap) merge(source stateMap) {
	for k, val := range source {
		state[k] = val
	}
}

func (state stateMap) applyIgnorefile(vfs afero.Fs, ignorefile string) ([]string, error) {
	// Apply nested ignorefile
	nestedIgnorefiles := []string{}
	for file := range state {
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
		nestedFileMap, errorFiles, err := createFileStateMap(nestedFs, ignorefile, false)
		if err != nil {
			return nil, err
		}
		for _, efile := range errorFiles {
			errorFiles = append(errorFiles, filepath.Join(nestedBasedir, efile))
		}

		for nfile, matched := range nestedFileMap {
			parentFile := filepath.Join(nestedBasedir, nfile)
			state[parentFile] = matched
		}
	}

	// Remove error files
	for _, efile := range errorFiles {
		errorFiles = append(errorFiles, efile)
		if ok := state[efile]; ok {
			delete(state, efile)
		}
	}

	return errorFiles, nil
}
