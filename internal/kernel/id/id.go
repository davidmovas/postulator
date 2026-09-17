package id

import "uuid"

const (
	canonicalLength = 36
	versionIndex    = 14
	variantIndex    = 19
)

var hyphenPositions = [4]int{8, 13, 18, 23}

func New() string {
	return uuid.NewV4().String()
}

func Valid(value string) bool {
	if len(value) != canonicalLength {
		return false
	}
	for _, position := range hyphenPositions {
		if value[position] != '-' {
			return false
		}
	}
	if value[versionIndex] != '4' {
		return false
	}
	switch value[variantIndex] {
	case '8', '9', 'a', 'A', 'b', 'B':
	default:
		return false
	}

	for i := range canonicalLength {
		if i == hyphenPositions[0] || i == hyphenPositions[1] || i == hyphenPositions[2] || i == hyphenPositions[3] {
			continue
		}
		if !isHex(value[i]) {
			return false
		}
	}

	_, err := uuid.Parse(value)
	return err == nil
}

func isHex(c byte) bool {
	switch {
	case c >= '0' && c <= '9':
		return true
	case c >= 'a' && c <= 'f':
		return true
	case c >= 'A' && c <= 'F':
		return true
	default:
		return false
	}
}
