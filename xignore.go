package xignore

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

// DirMatches returns match result from basedir.
func DirMatches(basedir string, options *MatchesOptions) (*MatchesResult, error) {
	return NewSystemMatcher().Matches(basedir, options)
}
