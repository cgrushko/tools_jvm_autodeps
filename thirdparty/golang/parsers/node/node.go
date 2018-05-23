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

// Package node provides types for working with AST nodes in a
// language-independent way.
package node

import (
	"fmt"

	lpb "github.com/bazelbuild/tools_jvm_autodeps/thirdparty/golang/parsers/lang"
)

// Type is the set of all AST node types returned by various language parsers.
// Most types correspond to nonterminal symbols from the respective language
// grammar.
//
// WARNING: do not change numeric assignments of the node types below!
type Type int16

// Selector represents a set of node types and returns true if a given type
// belongs to the set.
type Selector func(t Type) bool

// Any is a Selector which always returns true.
func Any(t Type) bool {
	return true
}

// Node types shared by all languages.
const (
	NoType        Type = 0
	BrokenFile    Type = 1
	SyntaxProblem Type = 2
	InvalidToken  Type = 3
	Keyword       Type = 4
	Punctuation   Type = 5
)

// Java
const (
	JavaFile               Type = 64
	JavaIdentifier         Type = 65 // a.foo
	JavaTraditionalComment Type = 66 // /* bb */ (including doc comments)
	JavaEndOfLineComment   Type = 67 // // aaa

	JavaPackage               Type = 68 // package com.test;
	JavaImport                Type = 69 // import a.b.c;
	JavaStatic                Type = 70 // import |static| A.b;
	JavaName                  Type = 71 // import |a.b.Cde|;
	JavaNameStar              Type = 72 // import |a.b.Cde.*|;
	JavaEmptyDecl             Type = 73 // ;
	JavaClass                 Type = 74 // class A {}
	JavaEnum                  Type = 75 // enum Kind {}
	JavaInterface             Type = 76 // interface Reader {}
	JavaAnnotationType        Type = 77 // public @interface Test { .. }
	JavaAnnotationTypeElement Type = 78 //    public boolean enabled() default true;

	JavaExtends               Type = 79  // class A |extends B| {}
	JavaImplements            Type = 80  // class D |implements E| {}
	JavaBody                  Type = 81  // class A |{}|
	JavaField                 Type = 82  // private int i = 1;
	JavaVarDecl               Type = 83  //             i = 1
	JavaMethod                Type = 84  // public static void main(String[] args) {}
	JavaFormalParameters      Type = 85  //                        (String[] args)
	JavaFormalParameter       Type = 86  //                         String[] args
	JavaMethodGenericClause   Type = 189 // public |<T> @X| T[] getTs();
	JavaVariadic              Type = 190 // void foo(String |@bar ...|rest)
	JavaReceiverParameter     Type = 87  // void m2(|@MyAnnotation Test this|) { }
	JavaThrows                Type = 88  // int a() |throws IOException| {}
	JavaNoBody                Type = 89  // class A { abstract int a()|;| }
	JavaEnumConstant          Type = 90  // enum I { |PUBLIC("public") { ... }| ... }
	JavaInstanceInitializer   Type = 91  // class A { |{}| }
	JavaStaticInitializer     Type = 92  // class A { |static {}| }
	JavaConstructor           Type = 93  // class A { |A() {}| }
	JavaIdentifierName        Type = 94  // class |A| {}
	JavaDefaultValue          Type = 95  // public boolean enabled() |default true|;
	JavaInitializerExpression Type = 96  // int i = |5+2|;
	JavaArrayInitializer      Type = 97  // byte[] bytes = |{1,2,3}|;

	JavaModifierKeyword  Type = 98  // public, final, @Nullable
	JavaAnnotation       Type = 99  // @Nullable
	JavaElementValuePair Type = 100 // @ForRequestPath(|value = "/aaa"|)
	JavaElementValue     Type = 101 // @ForRequestPath(|"/aaa"|)
	JavaLiteral          Type = 102 // 1, "abc", '\n', 1e9, 0xabc, 123_456

	JavaVoidType       Type = 103 // void
	JavaPrimitiveType  Type = 104 // int, long, boolean
	JavaClassType      Type = 105 // A, java.util.List<Integer>
	JavaClassTypeMods  Type = 188 // extends |@Bar| Foo {}
	JavaArrayType      Type = 106 // A[], int[]
	JavaTypeParameters Type = 107 // class A<|T extends A & B|> {}
	JavaTypeParameter  Type = 108 //          T extends A & B
	JavaTypeBound      Type = 109 //            extends A & B
	JavaTypeArguments  Type = 110 // Map|<String, List|<String>|>|
	JavaTypeArgument   Type = 111 // Map<|? extends String|, |B|>
	JavaBoundType      Type = 112 //        extends

	JavaBlock                 Type = 113 // { foo(); }
	JavaLocalVars             Type = 114 // final int a = 5, b = 7;
	JavaEmptyStatement        Type = 115 // ;
	JavaLabeled               Type = 116 // label: for (...) {}
	JavaExpressionStatement   Type = 117 // |a--;|
	JavaIf                    Type = 118 // if (true) { } else { }
	JavaAssert                Type = 119 // assert a == 5;
	JavaSwitch                Type = 120 // switch (a) { case 1: break; }
	JavaSwitchBlock           Type = 121 // switch (a) |{ case 1: break; }|
	JavaCase                  Type = 122 //              case 1:
	JavaDefaultCase           Type = 123 // switch (a) { |default:| break; }
	JavaWhile                 Type = 124 // while (true) { .. }
	JavaDoWhile               Type = 125 // do retry = run(); while(retry);
	JavaBasicFor              Type = 126 // for (int i = 0; i < arr.length; i++) { }
	JavaForInit               Type = 127 //      int i = 0
	JavaForUpdate             Type = 128 //                                 i++
	JavaEnhFor                Type = 129 // for (A a : listOfA) { }
	JavaBreak                 Type = 130 // break;
	JavaContinue              Type = 131 // continue A;
	JavaReturn                Type = 132 // return 1;
	JavaThrow                 Type = 133 // throw new IOException('failure');
	JavaSynchronized          Type = 134 // synchronized (a) { .. }
	JavaTryStatement          Type = 135 // try { } finally { } catch (IOException e) { .. }
	JavaFinally               Type = 136 //         finally { }
	JavaCatch                 Type = 137 //                     catch (IOException e) { .. }
	JavaCatchParameter        Type = 138 //                            IOException e
	JavaResourceSpecification Type = 139 // try |(InputStream inputStream = getStream(); )| { }
	JavaResource              Type = 140 //       InputStream inputStream = getStream()

	JavaMethodName       Type = 141 // this.|aa|(foo, bar)
	JavaArgs             Type = 187 //         |(foo, bar)|
	JavaThisCall         Type = 142
	JavaSuperCall        Type = 143
	JavaThis             Type = 144 // this
	JavaParenthesized    Type = 145
	JavaClassLiteral     Type = 146
	JavaQualifiedNew     Type = 147
	JavaNew              Type = 148
	JavaFieldAccess      Type = 149
	JavaArrayAccess      Type = 150
	JavaMethodInvocation Type = 151
	JavaSuperRef         Type = 152
	JavaMethodReference  Type = 153
	JavaNewArray         Type = 154
	JavaDimExpr          Type = 155
	JavaCastExpression   Type = 156
	JavaLambda           Type = 157
	JavaLambdaParameters Type = 158
	JavaAssignment       Type = 159
	JavaAssignmentOp     Type = 160
	JavaTernary          Type = 161
	JavaOr               Type = 162
	JavaAnd              Type = 163
	JavaBitOr            Type = 164
	JavaBitXor           Type = 165
	JavaBitAnd           Type = 166
	JavaEquality         Type = 167
	JavaInequality       Type = 168
	JavaRelational       Type = 169
	JavaInstanceOf       Type = 170
	JavaShift            Type = 171
	JavaAdditive         Type = 172
	JavaMultiplicative   Type = 173
	JavaUnary            Type = 174
	JavaPreInc           Type = 175
	JavaPreDec           Type = 176
	JavaPostInc          Type = 177
	JavaPostDec          Type = 178

	JavaTypeName       Type = 179
	JavaExprName       Type = 180
	JavaDim            Type = 181
	JavaTypeOrExprName Type = 182

	JavaModuleDeclaration Type = 183
	JavaModuleName        Type = 184
	JavaPackageName       Type = 185
	JavaModuleDirective   Type = 186

	JavaNodeMax Type = 191
)

// Property is a bitmask of node properties.
type Property int

// The following list of properties should never be persisted and can be changed
// as needed.
const (
	// HasName nodes have names as part of their source ranges.
	HasName Property = iota << 1
	// IsReference nodes represent references to other source elements.
	IsReference
	// IsLocal is set for nodes that can be dropped from the declarative AST.
	// Example: method implementations are local, since any declarations in them are
	// not accessible from the outside.
	IsLocal
	// RetainsText is set for nodes that have their text stored in a declarative AST.
	RetainsText
	// AttractsComments is set for nodes that can have comments attached to them.
	AttractsComments
	// IsMultilineToken is set for nodes that may span multiple lines (e.g. JavaTraditionalComment).
	IsMultilineToken
	// Identifier, Keywords, and other tokens that can be considered for token-based completion.
	IdentifierLike
)

// Category is a language-independent classification of node types. Each type
// gets mapped to the most specific category.
type Category int

// The following list of categories should never be persisted and can be changed
// as needed.
const (
	NoCategory Category = iota

	PackageClause
	ImportClause
	ExportClause
	Comment
	Name
	Identifier // for identifiers that are not names, types, parameters, or modifier
	Literal
	Expr // does not cover literals, references, and identifiers
	Statement
	TypeExpr // can also be used for single identifiers
	Parameter
	Annotation
	Modifier // public, static, final
	ClassDecl
	FunctionDecl
	VariableDecl
	Declaration     // other declarations that are not classes, functions, or variables
	DeclarationPart // for throws, extends, etc clauses
)

// Language returns the node language.
func (t Type) Language() lpb.Language {
	switch {
	case JavaFile <= t && t < JavaNodeMax:
		return lpb.Language_JAVA
	}
	return lpb.Language_UNKNOWN_LANG
}

// Category returns the closest category for the given type.
func (t Type) Category() Category {
	// TODO: implement
	switch t {
	case JavaTraditionalComment, JavaEndOfLineComment:
		return Comment
	}
	return NoCategory
}

// HasProperty return true if the type has the given property set.
func (t Type) HasProperty(p Property) bool {
	// TODO: implement
	switch p {
	case AttractsComments:
		switch t {
		case JavaClass, JavaEnum, JavaInterface, JavaConstructor, JavaMethod:
			return true
		}
	case IsMultilineToken:
		switch t {
		// TODO: add multi-line string literals
		case JavaTraditionalComment:
			return true
		}
	case IdentifierLike:
		switch t {
		case Keyword, JavaIdentifier:
			return true
		}
	case IsLocal:
		switch t {
		case
			// Java
			JavaBlock, JavaTraditionalComment, JavaEndOfLineComment, JavaDefaultValue,
			JavaInitializerExpression, JavaArrayInitializer, JavaAnnotation, JavaLiteral:
			return true
		}
	case RetainsText:
		switch t {
		case JavaIdentifier:
			return true
		}
	}
	return false
}

var typeName = map[Type]string{
	BrokenFile:    "BrokenFile",
	SyntaxProblem: "SyntaxProblem",
	InvalidToken:  "InvalidToken",
	Keyword:       "Keyword",
	Punctuation:   "Punctuation",
}

func (t Type) String() string {
	if name, ok := typeName[t]; ok {
		return name
	}
	return fmt.Sprintf("Unknown(%d)", t)
}

// OneOf constructs and returns a selector for the given set of node types.
func OneOf(types ...Type) Selector {
	if len(types) == 0 {
		return func(Type) bool { return false }
	}
	var max uint
	for _, t := range types {
		if uint(t) > max {
			max = uint(t)
		}
	}
	const bits = 32
	size := 1 + max/bits
	bitarr := make([]int32, size)
	for _, t := range types {
		bitarr[uint(t)/bits] |= 1 << (uint(t) % bits)
	}
	return func(t Type) bool {
		i := uint(t)
		return i <= max && bitarr[i/bits]&(1<<(i%bits)) != 0
	}
}

// TypeDescriptor contains all meta-information about one Type, provided by
// language implementations.
type TypeDescriptor struct {
	Type
	Name string
	// TODO: add fields for category and properties
}

// RegisterTypes propagates node types metadata for the given language to the
// global registry.
func RegisterTypes(l lpb.Language, types []TypeDescriptor) error {
	for _, td := range types {
		if td.Type.Language() != l {
			continue
		}
		if _, found := typeName[td.Type]; found {
			return fmt.Errorf("%s is registered twice", td.Name)
		}
		typeName[td.Type] = td.Name
	}
	return nil
}
