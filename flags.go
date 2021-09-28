package tracegen

import (
	"os"
	"regexp"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

type Settings struct {
	Exclude []string

	Tagged   bool
	Exported bool
	Methods  bool

	excludePatterns []*regexp.Regexp
}

func (s *Settings) Parse() (err error) {
	for _, pattern := range s.Exclude {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return errors.Wrapf(err, "invalid exclude pattern: %q", pattern)
		}

		s.excludePatterns = append(s.excludePatterns, re)
	}

	return nil
}

func DefaultSettings() (s Settings) {
	s.Exclude = []string{`/cmd(/|$)`}

	return s
}

func DefaultFlags(s *Settings) (flags *pflag.FlagSet) {
	if s == nil {
		settings := DefaultSettings()
		s = &settings
	}

	flags = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)

	flags.StringSliceVar(&s.Exclude, "exclude", s.Exclude, "if specified, do not run on matching packages")
	flags.BoolVar(&s.Tagged, "tagged", s.Tagged, "if specified, only run on tagged types, functions, and methods")
	flags.BoolVar(&s.Exported, "exported", s.Exported, "if specified, only run on exported types, functions, and methods")
	flags.BoolVar(&s.Methods, "methods", s.Methods, "if specified, only run on methods")

	return flags
}
