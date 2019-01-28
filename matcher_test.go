package xignore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatches_Simple(t *testing.T) {
	matcher := NewSystemMatcher()
	result, err := matcher.Matches("testdata/simple", &MatchesOptions{
		Ignorefile: ".xignore",
	})
	require.NoError(t, err)

	assert.Equal(t, []string{".xignore", "empty.log"}, result.MatchedFiles)
	assert.Equal(t, []string{"rain.txt"}, result.UnmatchedFiles)
	assert.Empty(t, result.ErrorFiles)
	assert.Empty(t, result.MatchedDirs)
	assert.Empty(t, result.UnmatchedDirs)
	assert.Empty(t, result.ErrorDirs)
}

func TestMatches_Root(t *testing.T) {
	matcher := NewSystemMatcher()
	result, err := matcher.Matches("testdata/root", &MatchesOptions{
		Ignorefile: ".xignore",
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"1.txt"}, result.MatchedFiles)
	assert.Equal(t, []string{".xignore", "sub/1.txt", "sub/2.txt"}, result.UnmatchedFiles)
	assert.Empty(t, result.ErrorFiles)
	assert.Empty(t, result.MatchedDirs)
	assert.Equal(t, result.UnmatchedDirs, []string{"sub"})
	assert.Empty(t, result.ErrorDirs)
}

func TestMatches_Exclusion(t *testing.T) {
	matcher := NewSystemMatcher()
	result, err := matcher.Matches("testdata/exclusion", &MatchesOptions{
		Ignorefile: ".xignore",
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"e1.txt", "e3.txt", "en/e3.txt"}, result.MatchedFiles)
	assert.Equal(t, []string{".xignore", "e2.txt", "en/e1.txt", "en/e2.txt"}, result.UnmatchedFiles)
	assert.Empty(t, result.ErrorFiles)
	assert.Empty(t, result.MatchedDirs)
	assert.Equal(t, result.UnmatchedDirs, []string{"en"})
	assert.Empty(t, result.ErrorDirs)
}
