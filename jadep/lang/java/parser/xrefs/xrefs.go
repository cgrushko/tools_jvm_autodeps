// Copyright 2018 The Jadep Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package xrefs provides support for local xrefs.
// For example, it resolves symbols in a single Java file.
package xrefs

import (
	"fmt"

	"github.com/bazelbuild/tools_jvm_autodeps/jadep/thirdparty/golang/parsers/ast"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/thirdparty/golang/parsers/node"
)

var typeSelector = node.OneOf(node.JavaClassType, node.JavaPrimitiveType, node.JavaArrayType)
var classTypeSelector = func(t node.Type) bool { return t == node.JavaClassType }

// SymbolTable describes what's in a container node, such as a class, enum, interface, method, etc.
type SymbolTable struct {
	Types   map[string]ast.Node   // <- interfaces, enums, classes, etc.
	Members map[string][]ast.Node // <- overloaded functions + fields with this name
}

func (st *SymbolTable) addType(n ast.Node) {
	if id := identifier(n); id != "" {
		st.Types[id] = n
	}
}

func (st *SymbolTable) addMember(n ast.Node) {
	if id := identifier(n); id != "" {
		st.Members[id] = append(st.Members[id], n)
	}
}

// identifier returns the first JavaIdentifierName child of 'n', which is its Java "name".
// For example, identifier(<JavaClass>) returns the class's (simple) name.
func identifier(n ast.Node) string {
	return n.FirstChildOfType(node.JavaIdentifierName).Text()
}

// walk walks an AST, calling 'before' on each node, and 'after' after all children have been visited.
// The return value of 'before' is passed to 'after'.
func walk(n ast.Node, before func(ast.Node) int, after func(ast.Node, int)) {
	x := before(n)
	for child := n.FirstChild(); child.IsValid(); child = child.NextSibling() {
		walk(child, before, after)
	}
	after(n, x)
}

func buildSymbolTables(t *ast.Tree) map[ast.Node]*SymbolTable {
	result := make(map[ast.Node]*SymbolTable)

	// stOfGrandparent returns the SymbolTable for the grand parent of 'n', or the tree root if there
	// isn't one.
	// All containers (e.g., JavaClass) have a JavaBody which contains members and inner types, so a
	// grandparent is the container of a member / inner type.
	stOfGrandparent := func(n ast.Node) *SymbolTable {
		container := n.Parent()
		if p := container.Parent(); p.IsValid() {
			container = p
		}
		if !isContainer(container.Type()) {
			panic(fmt.Sprintf("isContainer(%v) = false, but we expected true. It should be updated to return true in this case. Debug: n == %v", container.Type(), n))
		}
		st, ok := result[container]
		if !ok {
			st = &SymbolTable{make(map[string]ast.Node), make(map[string][]ast.Node)}
			result[container] = st
		}
		return st
	}

	before := func(n ast.Node) int {
		switch n.Type() {
		case node.JavaMethod, node.JavaEnumConstant:
			st := stOfGrandparent(n)
			st.addMember(n)

		case node.JavaField:
			st := stOfGrandparent(n)
			for child := n.FirstChild(); child.IsValid(); child = child.NextSibling() {
				if child.Type() == node.JavaVarDecl {
					st.addMember(child)
				}
			}

		case node.JavaClass, node.JavaEnum, node.JavaInterface, node.JavaTypeParameter, node.JavaAnnotationType:
			st := stOfGrandparent(n)
			st.addType(n)

		case node.JavaImport:
			static := n.FirstChildOfType(node.JavaStatic).IsValid()
			nameNode := n.FirstChildOfType(node.JavaName)
			if !nameNode.IsValid() {
				break
			}
			lastID := nameNode.LastChildOfType(node.JavaIdentifier)
			if !lastID.IsValid() {
				break
			}
			st := stOfGrandparent(n)
			name := lastID.Text()
			if static {
				st.Members[name] = append(st.Members[name], n)
			} else {
				st.Types[name] = n
			}
		}
		return 0
	}

	walk(t.Root(), before, func(n ast.Node, i int) {})
	return result
}

// Resolver resolves symbols in a Java program.
// Each identifier is mapped to the ast.Node that defined it.
type Resolver struct {
	tree         *ast.Tree
	SymbolTables map[ast.Node]*SymbolTable
}

// NewResolver returns a new Resolver based on an AST.
func NewResolver(tree *ast.Tree) *Resolver {
	return &Resolver{
		tree:         tree,
		SymbolTables: buildSymbolTables(tree),
	}
}

// Resolve maps symbols in the AST to the nodes that define them.
// For example, a field in an expression will be mapped to the field definition node.
func (r *Resolver) Resolve() map[ast.Node]ast.Node {
	bindings := make(map[ast.Node]ast.Node)
	r.resolveTypes(bindings)
	r.resolveNonTypes(bindings)
	return bindings
}

// isContainer returns true if 'n' is a type which can have a symbol table attached to it.
// It is used to avoid unnecessary hash table lookups in resolve* methods.
func isContainer(n node.Type) bool {
	switch n {
	case node.JavaAnnotationType,
		node.JavaConstructor,
		node.JavaEnum,
		node.JavaEnumConstant,
		node.JavaFile,
		node.JavaInterface,
		node.JavaMethod,
		node.JavaNew,
		node.JavaClass:
		return true
	}
	return false
}

// resolveTypes builds a map from usages of types (e.g. new Foo()) to their definitions
// (e.g. class Foo { ... }).
// The keys of the resulting map are the individual node.JavaIdentifier nodes.
// Only nodes which are definitely types (i.e. node.JavaTypeName) are mapped; all other types
// are mapped in resolveNonTypes().
//
// bindings is used to return the output.
func (r *Resolver) resolveTypes(bindings map[ast.Node]ast.Node) {
	var stack []*SymbolTable
	before := func(n ast.Node) int {
		pushedST := 0
		if isContainer(n.Type()) {
			if st := r.SymbolTables[n]; st != nil {
				stack = append(stack, st)
				pushedST = 1
			}
		}
		switch n.Type() {
		case node.JavaTypeName:
			ids := n.ChildrenOfType(node.JavaIdentifier)
			typeNode := typeInScope(stack, ids[0].Text())
			if typeNode.IsValid() {
				bindings[ids[0]] = typeNode
				r.resolveTypeChain(typeNode, ids[1:], bindings)
			}
		}
		return pushedST
	}

	after := func(n ast.Node, pushedST int) {
		stack = stack[:len(stack)-pushedST]
	}

	walk(r.tree.Root(), before, after)
}

// resolveNonTypes builds a map from usages of symbols (e.g. expressions) to their definitions
// (e.g. class Foo { ... }).
// The keys of the resulting map are the individual node.JavaIdentifier nodes.
// This method is called after resolveTypes(), and maps most symbols.
//
// bindings is used both to resolve symbols, and to return the result.
func (r *Resolver) resolveNonTypes(bindings map[ast.Node]ast.Node) {
	var stack []*SymbolTable
	before := func(n ast.Node) int {
		pushedST := 0
		if isContainer(n.Type()) {
			if st := r.SymbolTables[n]; st != nil {
				stack = append(stack, st)
				pushedST = 1
			}
		}
		switch n.Type() {
		case node.JavaBlock, node.JavaSwitchBlock, node.JavaBasicFor, node.JavaEnhFor:
			stack = append(stack, &SymbolTable{Members: make(map[string][]ast.Node)})
			pushedST++

		case node.JavaLocalVars, node.JavaForInit:
			st := stack[len(stack)-1]
			for child := n.FirstChild(); child.IsValid(); child = child.NextSibling() {
				if child.Type() == node.JavaVarDecl {
					st.addMember(child)
				}
			}

		case node.JavaTypeOrExprName, node.JavaExprName:
			ids := n.ChildrenOfType(node.JavaIdentifier)
			id0Text := ids[0].Text()
			if typ := typeInScope(stack, id0Text); typ.IsValid() {
				bindings[ids[0]] = typ
				lastIdx, lastTyp := r.resolveTypeChain(typ, ids[1:], bindings)
				r.resolveIdentifierChain(lastTyp, ids[1+lastIdx:], bindings)
				break
			}
			if field := fieldInScope(stack, id0Text); field.IsValid() {
				bindings[ids[0]] = field
				r.resolveIdentifierChain(field, ids[1:], bindings)
				break
			}
		}
		return pushedST
	}

	after := func(n ast.Node, pushedST int) {
		stack = stack[:len(stack)-pushedST]
	}

	walk(r.tree.Root(), before, after)
}

// typeInScope returns a type named 'id' in the closest lexical scope.
// 'scopes' is a stack of symbol tables, the last entry being the closest to us.
// If there's no such type, returns an invalid node.
func typeInScope(scopes []*SymbolTable, id string) ast.Node {
	for i := len(scopes) - 1; i >= 0; i-- {
		if n, ok := scopes[i].Types[id]; ok {
			return n
		}
	}
	return ast.Node{}
}

// fieldInScope returns a field named 'id' in the closest lexical scope.
// 'scopes' is a stack of symbol tables, the last entry being the closest to us.
// If there's no such field, returns an invalid node.
func fieldInScope(scopes []*SymbolTable, id string) ast.Node {
	for i := len(scopes) - 1; i >= 0; i-- {
		for _, n := range scopes[i].Members[id] {
			if n.Type() == node.JavaVarDecl || n.Type() == node.JavaImport {
				return n
			}
		}
	}
	return ast.Node{}
}

// resolveTypeChain resolves a chain of types, e.g. Foo.Bar.Baz, such that each identifier is
// mapped to its definition node.
// typeDeclaration is the first entry (Foo), and identifiers is a slice of
// node.JavaIdentifier (Bar, Baz).
// Bar is mapped to an inner type of Foo, and Baz to the inner type of Bar.
// The result is written into bindings.
func (r *Resolver) resolveTypeChain(typeDeclaration ast.Node, identifiers []ast.Node, bindings map[ast.Node]ast.Node) (int, ast.Node) {
	curr := typeDeclaration
	i := 0
	for ; i < len(identifiers); i++ {
		id := identifiers[i]
		st := r.SymbolTables[curr]
		if st == nil {
			break
		}
		if n, ok := st.Types[id.Text()]; ok {
			bindings[id] = n
			curr = n
		} else {
			break
		}
	}
	return i, curr
}

// resolveIdentifierChain resolves a chain of fields, e.g. Foo.a.b, such that each identifier is
// mapped to its definition node.
// declaration is the first entry (Foo) which can be either a field or a type, and identifiers is
// a slice of node.JavaIdentifier (a, b).
// 'a' is mapped to a field of Foo named 'a', and 'b' to the field of the type of 'a'.
// The result is written into bindings.
func (r *Resolver) resolveIdentifierChain(declaration ast.Node, identifiers []ast.Node, bindings map[ast.Node]ast.Node) {
	curr := declaration
idloop:
	for _, id := range identifiers {
		st := r.SymbolTables[typeBinding(curr, bindings)]
		if st == nil {
			break
		}
		for _, m := range st.Members[id.Text()] {
			if m.Type() == node.JavaVarDecl {
				fieldType := m.Prev(typeSelector)
				if !fieldType.IsValid() {
					continue
				}
				bindings[id] = m

				if fieldType.Type() == node.JavaClassType {
					decl := bindings[m.Prev(classTypeSelector).FirstChildOfType(node.JavaTypeName).LastChildOfType(node.JavaIdentifier)]
					if decl.IsValid() {
						curr = decl
						continue idloop
					}
				}
				return
			}
		}
		return
	}
}

// typeBinding returns the node declaring the type that 'n' is of, or an invalid node if it's
// unknown.
// For example, if 'n' is a field of type Foo, then typeBinding(n) = node that declares Foo.
func typeBinding(n ast.Node, bindings map[ast.Node]ast.Node) ast.Node {
	switch n.Type() {
	case node.JavaClass, node.JavaEnum, node.JavaInterface:
		return n
	case node.JavaClassType:
		return bindings[n.FirstChildOfType(node.JavaTypeName).LastChildOfType(node.JavaIdentifier)]
	case node.JavaVarDecl:
		return typeBinding(n.Prev(classTypeSelector), bindings)
	}
	return ast.Node{}
}
