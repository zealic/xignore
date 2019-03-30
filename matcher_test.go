package xignore

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func PathEquals(t *testing.T, expected []string, actual []string) {
	normalizedPaths := []string{}
	for _, p := range expected {
		normalizedPaths = append(normalizedPaths, filepath.FromSlash(p))
	}

	if len(normalizedPaths) == 0 {
		require.Empty(t, actual)
	} else {
		require.Equal(t, normalizedPaths, actual)
	}
}

func TestMatches_Simple(t *testing.T) {
	matcher := NewSystemMatcher()
	result, err := matcher.Matches("testdata/simple", &MatchesOptions{
		Ignorefile: ".xignore",
	})
	require.NoError(t, err)

	PathEquals(t, []string{".xignore", "empty.log"}, result.MatchedFiles)
	PathEquals(t, []string{"rain.txt"}, result.UnmatchedFiles)
	PathEquals(t, []string{}, result.MatchedDirs)
	PathEquals(t, []string{}, result.UnmatchedDirs)
}

func TestMatches_Simple_WithBeforePatterns(t *testing.T) {
	matcher := NewSystemMatcher()
	result, err := matcher.Matches("testdata/simple", &MatchesOptions{
		Ignorefile:     ".xignore",
		BeforePatterns: []string{"rain.txt"},
	})
	require.NoError(t, err)

	PathEquals(t, []string{".xignore", "empty.log", "rain.txt"}, result.MatchedFiles)
	PathEquals(t, []string{}, result.UnmatchedFiles)
	PathEquals(t, []string{}, result.MatchedDirs)
	PathEquals(t, []string{}, result.UnmatchedDirs)
}

func TestMatches_Simple_WithAfterPatterns(t *testing.T) {
	matcher := NewSystemMatcher()
	result, err := matcher.Matches("testdata/simple", &MatchesOptions{
		Ignorefile:    ".xignore",
		AfterPatterns: []string{"!.xignore", "!empty.log", "rain.txt"},
	})
	require.NoError(t, err)

	PathEquals(t, []string{"rain.txt"}, result.MatchedFiles)
	PathEquals(t, []string{".xignore", "empty.log"}, result.UnmatchedFiles)
	PathEquals(t, []string{}, result.MatchedDirs)
	PathEquals(t, []string{}, result.UnmatchedDirs)
}

func TestMatches_Folder(t *testing.T) {
	matcher := NewSystemMatcher()
	result, err := matcher.Matches("testdata/folder", &MatchesOptions{
		Ignorefile: ".xignore",
	})
	require.NoError(t, err)

	PathEquals(t, []string{"foo/bar/1.txt"}, result.MatchedFiles)
	PathEquals(t, []string{".xignore", "foo/bar/tool/lex.txt", "foo/tar/2.txt"}, result.UnmatchedFiles)
	PathEquals(t, []string{"foo/bar"}, result.MatchedDirs)
	PathEquals(t, []string{"foo", "foo/bar/tool", "foo/tar"}, result.UnmatchedDirs)
}

func TestMatches_Root(t *testing.T) {
	matcher := NewSystemMatcher()
	result, err := matcher.Matches("testdata/root", &MatchesOptions{
		Ignorefile: ".xignore",
	})
	require.NoError(t, err)

	PathEquals(t, []string{"1.txt"}, result.MatchedFiles)
	PathEquals(t, []string{".xignore", "sub/1.txt", "sub/2.txt"}, result.UnmatchedFiles)
	PathEquals(t, []string{}, result.MatchedDirs)
	PathEquals(t, []string{"sub"}, result.UnmatchedDirs)
}

func TestMatches_Exclusion(t *testing.T) {
	matcher := NewSystemMatcher()
	result, err := matcher.Matches("testdata/exclusion", &MatchesOptions{
		Ignorefile: ".xignore",
	})
	require.NoError(t, err)

	PathEquals(t, []string{"e1.txt", "e3.txt", "en/e3.txt"}, result.MatchedFiles)
	PathEquals(t, []string{"!", ".xignore", "e2.txt", "en/e1.txt", "en/e2.txt"}, result.UnmatchedFiles)
	PathEquals(t, []string{}, result.MatchedDirs)
	PathEquals(t, []string{"en"}, result.UnmatchedDirs)
}

func TestMatches_DisabledNested(t *testing.T) {
	matcher := NewSystemMatcher()
	result, err := matcher.Matches("testdata/nested", &MatchesOptions{
		Ignorefile: ".xignore",
		Nested:     false,
	})
	require.NoError(t, err)

	PathEquals(t, []string{
		"inner/foo.md",
	}, result.MatchedFiles)
	PathEquals(t, []string{
		".xignore", "1.txt",
		"inner/.xignore", "inner/2.lst",
		"inner/inner2/.xignore", "inner/inner2/jess.ini", "inner/inner2/moss.ini",
	}, result.UnmatchedFiles)
	PathEquals(t, []string{}, result.MatchedDirs)
	PathEquals(t, []string{"inner", "inner/inner2"}, result.UnmatchedDirs)
}

func TestMatches_Nested(t *testing.T) {
	matcher := NewSystemMatcher()
	result, err := matcher.Matches("testdata/nested", &MatchesOptions{
		Ignorefile: ".xignore",
		Nested:     true,
	})
	require.NoError(t, err)

	PathEquals(t, []string{
		"inner/2.lst", "inner/foo.md", "inner/inner2/moss.ini",
	}, result.MatchedFiles)
	PathEquals(t, []string{
		".xignore", "1.txt",
		"inner/.xignore",
		"inner/inner2/.xignore", "inner/inner2/jess.ini",
	}, result.UnmatchedFiles)
	PathEquals(t, []string{}, result.MatchedDirs)
	PathEquals(t, []string{"inner", "inner/inner2"}, result.UnmatchedDirs)
}

func TestMatches_ByName(t *testing.T) {
	matcher := NewSystemMatcher()
	result, err := matcher.Matches("testdata/byname", &MatchesOptions{
		Ignorefile: ".xignore",
	})
	require.NoError(t, err)

	PathEquals(t, []string{
		"aa/a1/a2/hello.txt", "aa/a1/hello.txt", "aa/hello.txt", "bb/hello.txt", "hello.txt",
	}, result.MatchedFiles)
	PathEquals(t, []string{".xignore"}, result.UnmatchedFiles)
	PathEquals(t, []string{}, result.MatchedDirs)
	PathEquals(t, []string{"aa", "aa/a1", "aa/a1/a2", "bb"}, result.UnmatchedDirs)
}

func TestMatches_Bothname(t *testing.T) {
	matcher := NewSystemMatcher()
	result, err := matcher.Matches("testdata/bothname", &MatchesOptions{
		Ignorefile: ".xignore",
	})
	require.NoError(t, err)

	PathEquals(t, []string{
		"foo/loss.txt", "loss.txt/1.log", "loss.txt/2.log",
	}, result.MatchedFiles)
	PathEquals(t, []string{".xignore"}, result.UnmatchedFiles)
	PathEquals(t, []string{"loss.txt"}, result.MatchedDirs)
	PathEquals(t, []string{"foo"}, result.UnmatchedDirs)
}

func TestMatches_LeadingSpace(t *testing.T) {
	matcher := NewSystemMatcher()
	result, err := matcher.Matches("testdata/leadingspace", &MatchesOptions{
		Ignorefile: ".xignore",
	})
	require.NoError(t, err)

	PathEquals(t, []string{
		"  what.txt",
		"inner2/  what.txt",
	}, result.MatchedFiles)
	PathEquals(t, []string{".xignore", "inner/  what.txt"}, result.UnmatchedFiles)
	PathEquals(t, []string{}, result.MatchedDirs)
	PathEquals(t, []string{"inner", "inner2"}, result.UnmatchedDirs)
}
