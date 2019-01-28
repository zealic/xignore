package xignore

// DirMatches returns match result from basedir.
func DirMatches(basedir string, options *MatchesOptions) (*MatchesResult, error) {
	return NewSystemMatcher().Matches(basedir, options)
}
