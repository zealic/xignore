package xignore

// Pattern defines a single regexp used used to filter file paths.
type Pattern struct {
	value     string
	exclusion bool
}

// NewPattern create new pattern
func NewPattern(strPattern string) *Pattern {
	if len(strPattern) == 0 {
		return &Pattern{value: ""} // empty
	}

	if strPattern[0] == '!' && len(strPattern) > 1 {
		strPattern = strPattern[1:]
		return &Pattern{value: strPattern, exclusion: true}
	}

	return &Pattern{value: strPattern}
}

func (p *Pattern) String() string {
	return p.value
}

// IsExclusion returns true if this pattern defines exclusion
func (p *Pattern) IsExclusion() bool {
	return p.exclusion
}

// IsEmpty returns true if this pattern is empty
func (p *Pattern) IsEmpty() bool {
	return p.value == ""
}
