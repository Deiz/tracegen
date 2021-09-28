package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Deiz/tracegen"
)

const gomod = `module test

go 1.17

require github.com/opentracing/opentracing-go v1.2.0`

const input0 = `package main

import "context"

func Foo(ctx context.Context) {}
`

const output0 = `package main

import (
	"context"

	"github.com/opentracing/opentracing-go"
)

func Foo(ctx context.Context) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Foo")
	defer span.Finish()
}
`

const input1 = output0
const output1 = input0

const input2 = `package main

import (
	"context"

	"github.com/opentracing/opentracing-go"
)

//trace:skip
func Foo(ctx context.Context) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Foo")
	defer span.Finish()
}
`

const output2 = `package main

import "context"

//trace:skip
func Foo(ctx context.Context) {}
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

func TestUpdater(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected string
		settings tracegen.Settings
	}{
		"add span":           {input0, output0, tracegen.Settings{}},
		"remove span":        {input1, output1, tracegen.Settings{Methods: true}},
		"remove span (skip)": {input2, output2, tracegen.Settings{}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			path := writeModule(t, test.input)

			err := os.Chdir(filepath.Dir(path))
			check(t, err)

			err = tracegen.Process(
				test.settings,
				[]string{"."},
				update,
				getResolver,
			)
			check(t, err)

			data, err := os.ReadFile(path)
			check(t, err)

			if string(data) != test.expected {
				t.Fatalf("mismatched output in %s:\ngot:\n%s\nexpected:\n%s", path, string(data), test.expected)
			}
		})
	}
}
