// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build go1.18
// +build go1.18

package main

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"log"
)

const hello = `
//!+input
package main

type Constraint interface {
	Value() any
}

type Pair[L, R any] struct {
	left  L
	right R
}

func MakePair[L, R Constraint](l L, r R) Pair[L, R] {
	return Pair[L, R]{l, r}
}
//!-input
`

//!+print
func PrintTypeParams(fset *token.FileSet, file *ast.File) error {
	conf := types.Config{Importer: importer.Default()}
	info := &types.Info{
		Scopes: make(map[ast.Node]*types.Scope),
		Defs:   make(map[*ast.Ident]types.Object),
	}
	_, err := conf.Check("hello", fset, []*ast.File{file}, info)
	if err != nil {
		return err
	}

	// For convenience, we can use ast.Inspect to find the nodes we want to
	// investigate.
	ast.Inspect(file, func(n ast.Node) bool {
		var name *ast.Ident              // the name of the generic object, or nil
		var tparamFields *ast.FieldList  // the list of type parameter fields
		var tparams *types.TypeParamList // the list of type parameter types
		var scopeNode ast.Node           // the node associated with the declaration scope

		switch n := n.(type) {
		case *ast.TypeSpec:
			name = n.Name
			tparamFields = n.TypeParams
			tparams = info.Defs[name].Type().(*types.Named).TypeParams()
			scopeNode = n
		case *ast.FuncDecl:
			name = n.Name
			tparamFields = n.Type.TypeParams
			tparams = info.Defs[name].Type().(*types.Signature).TypeParams()
			scopeNode = n.Type
		default:
			// Not a type or function declaration.
			return true
		}

		// Option 1: find type parameters by looking at their declaring field list.
		if tparamFields != nil {
			fmt.Printf("%s has a type parameter field list with %d fields\n", name.Name, tparamFields.NumFields())
			for _, field := range tparamFields.List {
				for _, name := range field.Names {
					tparam := info.Defs[name]
					fmt.Printf("  field %s defines an object %q\n", name.Name, tparam)
				}
			}
		} else {
			fmt.Printf("%s does not have a type parameter list\n", name.Name)
		}

		// Option 2: find type parameters via the TypeParams() method on the
		// generic type.
		if tparams.Len() > 0 {
			fmt.Printf("%s has %d type parameters:\n", name.Name, tparams.Len())
			for i := 0; i < tparams.Len(); i++ {
				tparam := tparams.At(i)
				fmt.Printf("  %s has constraint %s\n", tparam, tparam.Constraint())
			}
		} else {
			fmt.Printf("%s does not have type parameters\n", name.Name)
		}

		// Option 3: find type parameters by looking in the declaration scope.
		scope, ok := info.Scopes[scopeNode]
		if ok {
			fmt.Printf("%s has a scope with %d objects:\n", name.Name, scope.Len())
			for _, name := range scope.Names() {
				fmt.Printf("  %s is a %T\n", name, scope.Lookup(name))
			}
		} else {
			fmt.Printf("%s does not have a scope\n", name.Name)
		}

		return true
	})
	return nil
}

//!-print

/*
//!+output
> go run golang.org/x/tools/internal/typeparams/example/findtypeparams
Constraint does not have a type parameter list
Constraint does not have type parameters
Constraint does not have a scope
Pair has a type parameter field list with 2 fields
  field L defines an object "type parameter L any"
  field R defines an object "type parameter R any"
Pair has 2 type parameters:
  L has constraint any
  R has constraint any
Pair has a scope with 2 objects:
  L is a *types.TypeName
  R is a *types.TypeName
MakePair has a type parameter field list with 2 fields
  field L defines an object "type parameter L hello.Constraint"
  field R defines an object "type parameter R hello.Constraint"
MakePair has 2 type parameters:
  L has constraint hello.Constraint
  R has constraint hello.Constraint
MakePair has a scope with 4 objects:
  L is a *types.TypeName
  R is a *types.TypeName
  l is a *types.Var
  r is a *types.Var
//!-output
*/

func main() {
	// Parse one file.
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "hello.go", hello, 0)
	if err != nil {
		log.Fatal(err)
	}
	if err := PrintTypeParams(fset, f); err != nil {
		log.Fatal(err)
	}
}
