# tracegen

tracegen is a utility that injects tracing code into functions and methods.

Its purpose is to ensure the presence of tracing code without adding a human maintenance cost, and also to reduce the chances of drift.

Its feature set is as follows:

- Supports opt-in and opt-out models for the injection of code into functions and methods
- Supports `//trace:skip` at the type, function, and method level
  - If applied to a type, all methods will be skipped by default
- Supports `//trace:enable` at the type, function, and method level
  - If applied to a type, all methods will be traced by default

The default `cmd/tracegen` updater targets functions and methods that have `context.Context` as their first parameter, and then ensures the beginning of the body resembles:

```go
func Foo(ctx context.Context) {
    span, ctx := opentracing.StartSpanFromContext(ctx, "Foo")
    defer span.Finish()

    // ...
}
```

## Installation

### CLI

```sh
go install github.com/Deiz/tracegen/cmd/...@latest
```

### Library

```sh
go get github.com/Deiz/tracegen
```

## Usage

## CLI

```sh
tracegen ./...
```

## Library

Using tracegen as a library requires you to implement an updater, as well as an import resolver.

tracegen is built on top of [github.com/dave/dst](https://github.com/dave/dst) and
exposes its primitives via the two functions that users of the library must implement.

### updater

The updater will be invoked for every function and method in the target package(s).

Correct, idempotent behaviour of tracegen is entirely dependent on the updater.

Generally speaking, an updater should:

- Inject its code when `shouldSkip` is false
- Remove its code (if present) when `shouldSkip` is true
- Ideally preserve comments and whitespace surrounding the updater-managed code
- Return all imports needed by the inserted code

```go
func (fn *dst.FuncDecl, shouldSkip bool) (imports []string)
```

See `cmd/tracegen` for a sample implementation.

### resolver

The resolver must resolve any existing import in the supplied package along
with any imports the resolver itself introduces.

```go
func (pkg *decorator.Package) resolver.RestorerResolver
```

See `cmd/tracegen` for a sample implementation.

### CLI

The tracegen library's settings struct is typically built using its standard
command-line flags, and the building blocks are exported as part of the library.

Once you've implemented `updater` and `resolver` functions, building a custom CLI
looks like this:

```go
func main() {
	settings := tracegen.DefaultSettings()
	flags := tracegen.DefaultFlags(&settings)

	if err := flags.Parse(os.Args[1:]); err != nil {
		log.Fatalf("failed to parse flags: %v", err)
	}

	if flags.NArg() < 1 {
		log.Fatal("must specify at least one pattern")
	}

	if err := settings.Parse(); err != nil {
		log.Fatalf("failed to parse settings: %v", err)
	}

	if err := tracegen.Process(settings, flags.Args(), updater, resolver); err != nil {
		log.Fatalf("failed to process: %v", err)
	}
}
```
