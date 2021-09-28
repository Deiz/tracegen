package tracegen

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/dave/dst/decorator/resolver"
)

const gomod = `module test

go 1.17`

const inputFunc = `package main

func Foo() {}
`

const inputMethod = `package main

type Foo struct{}

func (f *Foo) Foo() {}
`

const inputNonExportedFunc = `package main

func foo() {}
`

const skippedFuncAndMethod = `package main

//trace:skip
func Foo() {}

type Bar struct{}

//trace:skip
func (b *Bar) Foo() {}
`

const explicitIncludeMethod = `package main

//trace:skip
type Foo struct{}

func (f *Foo) A() {}

//trace:enable
func (f *Foo) B() {}
`

func check(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatal(err)
	}
}

func writeModule(t *testing.T, sample string) (path string) {
	t.Helper()

	dir, err := os.MkdirTemp("", "")
	check(t, err)

	path = filepath.Join(dir, "sample.go")
	err = os.WriteFile(path, []byte(sample), 0644)
	check(t, err)

	err = os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0644)
	check(t, err)

	return path
}

func TestProcess(t *testing.T) {
	tests := map[string]struct {
		input    string
		calls    []bool
		settings Settings
	}{
		"default calls funcs":                                {inputFunc, []bool{false}, Settings{}},
		"default calls methods":                              {inputMethod, []bool{false}, Settings{}},
		"default calls non-exported funcs":                   {inputNonExportedFunc, []bool{false}, Settings{}},
		"default skips explicitly skipped funcs and methods": {skippedFuncAndMethod, []bool{true, true}, Settings{}},
		"methods skips funcs":                                {inputFunc, []bool{true}, Settings{Methods: true}},
		"exported skips non-exported funcs":                  {inputNonExportedFunc, []bool{true}, Settings{Exported: true}},
		"explicit include preempts exclude":                  {explicitIncludeMethod, []bool{true, false}, Settings{}},
		"explicit include preempts untagged parent":          {explicitIncludeMethod, []bool{true, false}, Settings{Tagged: true}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			path := writeModule(t, test.input)
			err := os.Chdir(filepath.Dir(path))
			check(t, err)

			var calls []bool

			err = Process(
				test.settings,
				[]string{"."},
				func(fn *dst.FuncDecl, shouldSkip bool) (imports []string) {
					calls = append(calls, shouldSkip)
					return nil
				},
				func(pkg *decorator.Package, file *dst.File) resolver.RestorerResolver {
					return NewSimpleResolver(pkg, file, nil)
				},
			)
			check(t, err)

			if !reflect.DeepEqual(calls, test.calls) {
				t.Fatalf("mismatched calls, got %v, expected %v", calls, test.calls)
			}
		})
	}
}
