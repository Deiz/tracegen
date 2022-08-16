package tracegen

import (
	"bytes"
	"go/token"
	"os"
	"path/filepath"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/dave/dst/decorator/resolver"
	"github.com/pkg/errors"
)

var (
	writer func(name string, data []byte, perm os.FileMode) error = os.WriteFile
)

// Process applies the specified update function to relevant functions discovered
// within packages matching the passed-in package patterns. The supplied resolver
// must be capable of matching any pre-existing import within the loaded packages
// as well as any introduced by the update function.
func Process(settings Settings, packages []string, update func(fn *dst.FuncDecl, shouldSkip bool) (imports []string), getResolver func(pkg *decorator.Package, file *dst.File) resolver.RestorerResolver) (err error) {
	pkgs, err := LoadPackages(packages)
	if err != nil {
		return
	}

	return ProcessPackages(settings, pkgs, update, getResolver)
}

func LoadPackages(packages []string) (pkgs []*decorator.Package, err error) {
	pkgs, err = decorator.Load(nil, packages...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load packages")
	}

	return pkgs, nil
}

func ProcessPackages(settings Settings, pkgs []*decorator.Package, update func(fn *dst.FuncDecl, shouldSkip bool) (imports []string), getResolver func(pkg *decorator.Package, file *dst.File) resolver.RestorerResolver) (err error) {
	for _, pkg := range pkgs {
		var excluded bool
		for _, pattern := range settings.excludePatterns {
			if pattern.MatchString(filepath.Join(pkg.Dir, pkg.Name)) {
				excluded = true
				break
			}
		}

		if excluded {
			continue
		}

		// Types to skip, based on trace:skip tags
		skipTypes := make(map[string]struct{})

		// Types to include, based on trace:enable tags
		enableTypes := make(map[string]struct{})

		for _, file := range pkg.Syntax {
			// Iterate through types first to build the skipTypes map
			dst.Inspect(file, func(n dst.Node) bool {
				switch node := n.(type) {
				case *dst.GenDecl:
					// Only process types
					if node.Tok != token.TYPE {
						return true
					}

					typeName := node.Specs[0].(*dst.TypeSpec).Name.Name
					if explicitInclude(node.Decs.Start) {
						enableTypes[typeName] = struct{}{}
						break
					}

					if skipByName(settings, typeName) {
						skipTypes[typeName] = struct{}{}
					} else if skipByComments(settings, node.Decs.Start) {
						skipTypes[typeName] = struct{}{}
					} else if settings.Tagged {
						skipTypes[typeName] = struct{}{}
					}
				}

				return true
			})
		}

		changed := make(map[string][]byte)

		for _, file := range pkg.Syntax {
			resolver := getResolver(pkg, file)

			pre, err := fileContents(pkg, file, resolver)
			if err != nil {
				return err
			}

			var skipped bool

			imports := make(map[string]struct{})

			// Iterate through functions next
			dst.Inspect(file, func(n dst.Node) bool {
				switch node := n.(type) {
				case *dst.FuncDecl:
					// Whether this function should be skipped
					var shouldSkip bool

					defer func() {
						if shouldSkip {
							skipped = true
						}
					}()

					// Whether this function should explicitly be included
					shouldInclude := !settings.Tagged

					if skipByName(settings, node.Name.Name) {
						shouldSkip = true
					}

					if skipByComments(settings, node.Decs.Start) {
						shouldSkip = true
					}

					// Check for a struct-level skip tag
					if node.Recv != nil {
						for _, field := range node.Recv.List {
							var typeName string

							switch field := field.Type.(type) {
							// Pointer receivers
							case *dst.StarExpr:
								ident, ok := field.X.(*dst.Ident)
								if !ok {
									shouldSkip = true
								} else {
									typeName = ident.Name
								}
							// Non-pointer receivers
							case *dst.Ident:
								typeName = field.Name
							}

							if typeName != "" {
								if _, include := enableTypes[typeName]; include {
									shouldInclude = true
								} else if _, skip := skipTypes[typeName]; skip {
									shouldSkip = true
								}
							}
						}
					} else if settings.Methods {
						shouldSkip = true
					}

					if explicitInclude(node.Decs.Start) {
						shouldSkip = false
					} else if settings.Tagged && !shouldSkip {
						shouldSkip = !shouldInclude
					}

					for _, imp := range update(node, shouldSkip) {
						imports[imp] = struct{}{}
					}
				}

				return true
			})

			if !skipped {
				for imp := range imports {
					addImport(pkg, file, imp)
				}
			}

			post, err := fileContents(pkg, file, resolver)
			if err != nil {
				return err
			}

			if !bytes.Equal(pre, post) {
				changed[pkg.Decorator.Filenames[file]] = post
			}
		}

		for filename, data := range changed {
			if err := writer(filename, data, 0666); err != nil {
				return errors.Wrapf(err, "failed to save file %s", filename)
			}
		}
	}

	return nil
}

func fileContents(p *decorator.Package, file *dst.File, resolver resolver.RestorerResolver) (data []byte, err error) {
	buf := &bytes.Buffer{}

	r := decorator.NewRestorerWithImports(p.PkgPath, resolver)
	if err := r.Fprint(buf, file); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
