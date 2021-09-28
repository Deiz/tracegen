package tracegen

import (
	"fmt"

	"github.com/dave/dst/decorator"
)

type SimpleResolver map[string]string

func NewSimpleResolver(pkg *decorator.Package, hints map[string]string) SimpleResolver {
	r := make(SimpleResolver)

	for name, pkg := range pkg.Imports {
		r[name] = pkg.Name
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
