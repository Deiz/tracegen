package tracegen

import (
	"fmt"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

type SimpleResolver map[string]string

func NewSimpleResolver(pkg *decorator.Package, file *dst.File, hints map[string]string) SimpleResolver {
	r := make(SimpleResolver)

	for name, pkg := range pkg.Imports {
		r[name] = pkg.Name
	}

	// To ensure we only resolve file-level imports, we omit any package-level
	// imports that aren't used in this file directly. This is because the pkg
	// has greater fidelity (its Imports attribute is map[string]*Package, and
	// the file's is []*ImportSpec) and we need a fully-qualified mapping from
	// e.g. github.com/pkg/errors => errors
	for _, imp := range file.Imports {
		path := imp.Path.Value
		if _, ok := r[path]; !ok {
			delete(r, path)
		}
	}

	for pkgPath, name := range hints {
		r[pkgPath] = name
	}

	return r
}

func (r SimpleResolver) ResolvePackage(importPath string) (string, error) {
	if n, ok := r[importPath]; ok {
		return n, nil
	}
	return "", fmt.Errorf("package %s was not found", importPath)
}
