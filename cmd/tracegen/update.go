package main

import (
	"fmt"
	"go/token"
	"strconv"

	"github.com/Deiz/tracegen"
	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/dave/dst/decorator/resolver"
)

var (
	hints = map[string]string{
		"github.com/opentracing/opentracing-go": "opentracing",
	}
)

func getResolver(pkg *decorator.Package) resolver.RestorerResolver {
	return tracegen.NewSimpleResolver(pkg, hints)
}

func update(fn *dst.FuncDecl, shouldSkip bool) (imports []string) {
	if !shouldSkip {
		imports = []string{"github.com/opentracing/opentracing-go"}
	}

	params := fn.Type.Params.List
	if len(params) == 0 || fmt.Sprintf("%s", params[0].Type) != "context.Context" {
		return
	}

	stmt := getStmt(fn)
	matched := make([]*int, len(stmt))

	getMatched := func() (r [][2]int) {
		for i, m := range matched {
			if m != nil {
				r = append(r, [2]int{i, *m})
			}
		}

		return r
	}

	defer func() {
		if !shouldSkip {
			// Try to avoid associating pre-existing comments with generated code
			// by adding a newline before the code.
			if len(fn.Body.Decs.Lbrace) > 0 && len(fn.Body.List) > len(getMatched()) {
				leadingNewline := fn.Body.Decs.Lbrace[0] == "\n"
				trailingNewline := fn.Body.Decs.Lbrace[len(fn.Body.Decs.Lbrace)-1] == "\n"

				if leadingNewline && !trailingNewline {
					fn.Body.Decs.Lbrace = append(fn.Body.Decs.Lbrace, "\n")
				}
			}

			// Idempotency
			for i, m := range matched {
				if m != nil {
					offset := *m - i
					dec := fn.Body.List[offset].Decorations()
					stmt[i].Decorations().Before = dec.Before
					stmt[i].Decorations().After = dec.After
					stmt[i].Decorations().Start = append(stmt[i].Decorations().Start, dec.Start...)
					stmt[i].Decorations().End = append(stmt[i].Decorations().End, dec.End...)

					// TODO(swh): This might not always be correct.
					if offset-1 >= 0 && fn.Body.List[offset-1].Decorations().After == dst.EmptyLine {
						fn.Body.List[offset-1].Decorations().After = dst.NewLine
					}

					// Remove the old statement
					fn.Body.List = append(fn.Body.List[:offset], fn.Body.List[offset+1:]...)
				}
			}

			// If there's a pre-existing function body, add an empty line after
			// the generated code.
			if len(stmt) > 0 && len(fn.Body.List) > 0 {
				stmt[len(stmt)-1].Decorations().After = dst.EmptyLine
			}

			// Ensure the generated code has an empty line above it
			if len(stmt) > 0 && stmt[0].Decorations().Before == dst.EmptyLine {
				stmt[0].Decorations().Before = dst.NewLine
			}

			fn.Body.List = append(stmt, fn.Body.List...)

			return
		}

		for offset, index := range matched {
			if index == nil {
				continue
			}

			i := *index - offset
			fn.Body.List = append(fn.Body.List[:i], fn.Body.List[i+1:]...)
		}

		if len(fn.Body.List) > 0 {
			fn.Body.List[0].Decorations().Before = dst.NewLine
		}
	}()

	if fn.Body == nil {
		return
	}

	for i, decl := range fn.Body.List {
		index := i
		switch stmt := decl.(type) {
		// Check for `span, ctx := opentracing.StartSpanFromContext`
		case *dst.AssignStmt:
			if len(stmt.Lhs) != 2 {
				// span, ctx
				continue
			} else if len(stmt.Rhs) != 1 {
				// opentracing.StartSpanFromContext
				continue
			}

			if ident, ok := stmt.Lhs[0].(*dst.Ident); !ok || ident.Name != "span" {
				continue
			} else if ident, ok := stmt.Lhs[1].(*dst.Ident); !ok || ident.Name != "ctx" {
				continue
			} else if stmt.Tok != token.DEFINE { // :=
				continue
			}

			if stmt, ok := stmt.Rhs[0].(*dst.CallExpr); ok {
				switch stmt := stmt.Fun.(type) {
				case *dst.SelectorExpr:
					if ident, ok := stmt.X.(*dst.Ident); ok && ident.Name != "github.com/opentracing/opentracing-go" && stmt.Sel.Name == "StartSpanFromContext" {
						matched[0] = &index
					}
				case *dst.Ident:
					if stmt.Path == "github.com/opentracing/opentracing-go" && stmt.Name == "StartSpanFromContext" {
						matched[0] = &index
					}
				}
			}
		// Check for `defer span.Finish()`
		case *dst.DeferStmt:
			if stmt, ok := stmt.Call.Fun.(*dst.SelectorExpr); ok {
				if ident, ok := stmt.X.(*dst.Ident); !ok || ident.Name != "span" {
					continue
				}

				if stmt.Sel.Name != "Finish" {
					continue
				}

				matched[1] = &index
			}
		}
	}

	return
}

func getStmt(fn *dst.FuncDecl) []dst.Stmt {
	return []dst.Stmt{
		&dst.AssignStmt{
			Lhs: []dst.Expr{
				&dst.Ident{
					Name: "span",
				},
				&dst.Ident{
					Name: "ctx",
				},
			},
			Tok: token.DEFINE,
			Rhs: []dst.Expr{
				&dst.CallExpr{
					Fun: &dst.Ident{
						Path: "github.com/opentracing/opentracing-go",
						Name: "StartSpanFromContext",
					},
					Args: []dst.Expr{
						&dst.Ident{Name: "ctx"},
						&dst.BasicLit{Kind: token.STRING, Value: strconv.Quote(fn.Name.Name)},
					},
				},
			},
		},
		&dst.DeferStmt{
			Call: &dst.CallExpr{
				Fun: &dst.SelectorExpr{
					X: &dst.Ident{
						Name: "span",
					},
					Sel: &dst.Ident{
						Name: "Finish",
					},
				},
			},
		},
	}
}
