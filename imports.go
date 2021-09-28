package tracegen

import (
	"fmt"
	"go/token"
	"strconv"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

func addImport(pkg *decorator.Package, file *dst.File, imp string) {
	for _, pkg := range pkg.Imports {
		if pkg.Name == imp {
			return
		}
	}

	// Where to insert our import block within the file's Decl slice
	index := 0

	importSpec := &dst.ImportSpec{
		Path: &dst.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("%q", imp)},
	}

	for i, node := range file.Decls {
		n, ok := node.(*dst.GenDecl)
		if !ok {
			continue
		}

		if n.Tok != token.IMPORT {
			continue
		}

		if len(n.Specs) == 1 && mustUnquote(n.Specs[0].(*dst.ImportSpec).Path.Value) == "C" {
			// If we're going to insert, it must be after the "C" import
			index = i + 1
			continue
		}

		// Insert our import into the first non-"C" import block
		for j, spec := range n.Specs {
			path := mustUnquote(spec.(*dst.ImportSpec).Path.Value)
			if !strings.Contains(path, ".") || imp > path {
				continue
			}

			importSpec.Decorations().Before = spec.Decorations().Before
			spec.Decorations().Before = dst.NewLine

			n.Specs = append(n.Specs[:j], append([]dst.Spec{importSpec}, n.Specs[j:]...)...)
			return
		}

		n.Specs = append(n.Specs, importSpec)
		return
	}

	gd := &dst.GenDecl{
		Tok:   token.IMPORT,
		Specs: []dst.Spec{importSpec},
		Decs: dst.GenDeclDecorations{
			NodeDecs: dst.NodeDecs{Before: dst.EmptyLine, After: dst.EmptyLine},
		},
	}

	file.Decls = append(file.Decls[:index], append([]dst.Decl{gd}, file.Decls[index:]...)...)
}

func mustUnquote(s string) string {
	out, err := strconv.Unquote(s)
	if err != nil {
		panic(err)
	}
	return out
}
