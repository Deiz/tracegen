package tracegen

import (
	"regexp"

	"github.com/dave/dst"
)

var (
	skipPattern    = regexp.MustCompile(`//\s*trace:skip`)
	includePattern = regexp.MustCompile(`//\s*trace:enable`)
)

func skipByName(c Settings, name string) bool {
	return c.Exported && !dst.IsExported(name)
}

func explicitInclude(decs []string) bool {
	for _, dec := range decs {
		if includePattern.MatchString(dec) {
			return true
		}
	}

	return false
}

func skipByComments(c Settings, decs []string) bool {
	if explicitInclude(decs) {
		return false
	}

	for _, dec := range decs {
		if skipPattern.MatchString(dec) {
			return true
		}
	}

	return false
}
