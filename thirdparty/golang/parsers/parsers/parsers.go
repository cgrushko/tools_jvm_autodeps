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

// Package parsers is the main entry point for lexing and parsing source code
// for IDE and presentation purposes.
package parsers

import (
	"errors"
	"fmt"

	lpb "github.com/bazelbuild/tools_jvm_autodeps/jadep/thirdparty/golang/parsers/lang"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/thirdparty/golang/parsers/node"
	tpb "github.com/bazelbuild/tools_jvm_autodeps/jadep/thirdparty/golang/parsers/public/token"
	"context"
)

// Version is the global version of the parsers toolkit, which advances every
// time we make an incompatible change in one of the parsers.
const Version = 1

var (
	// ErrUnsupportedLanguage indicates that the given language is not supported yet.
	ErrUnsupportedLanguage = errors.New("unsupported language")
)

// SyntaxError wraps low-level parsing errors and points to the first token
// that was not consumed by the parser.
type SyntaxError struct {
	Description string
	Line        int
	Offset      int
	Length      int
}

func (se SyntaxError) Error() string {
	return fmt.Sprintf("%d: %s", se.Line, se.Description)
}

// LexerListener receives all non-whitespace tokens of the language (including
// comments) in the order of their appearance in the input string.
//
// The given ranges are non-empty and never overlap (may touch though).
type LexerListener func(t tpb.TokenType, offset, endoffset int)

// HasLexer returns whether the language has a registered lexer.
func HasLexer(lang lpb.Language) bool {
	_, ok := lexers[lang]
	return ok
}

// Lex tokenizes "source" in the given language. It returns tokens from a
// unified set of token classes shared by all languages and can be used for
// syntax highlighting, detecting non-whitespace changes in a file, or
// enumerating all tokens of a certain class in a language-independent way.
//
// We expect all lexers to be cheap and process the input at least at 100MB/s.
// The function returns an error only if no lexer is registered for the language.
func Lex(ctx context.Context, lang lpb.Language, source string, listener LexerListener) error {
	if lexer, ok := lexers[lang]; ok {
		lexer(ctx, source, listener)
		return nil
	}
	return ErrUnsupportedLanguage
}

// ErrorHandler is a function which receives all non-fatal parser errors and decides whether we
// should continue parsing. If it returns false, the last error gets returned as the main outcome
// from the parser.
type ErrorHandler func(err SyntaxError) bool

// Options contains parameters that control parsing behavior.
type Options struct {
	// Adds node.Punctuation and node.Keyword nodes to the parse tree.
	IncludeAllTokens bool

	// A callback function which decides whether the parser should try to recover and continue
	// parsing. Successful error recovery leads to one or more SyntaxProblem or InvalidToken nodes
	// in the tree.
	//
	// Never called on syntactically valid input.
	// Leave unset to disable error recovery.
	ShouldTryToRecover ErrorHandler
}

// ParserListener gets all parsed source ranges in the left-to-right and
// parent-after-children order. Any two of the reported ranges either don't
// overlap, or contain one another.
//
// For some types, the range can be empty, indicating the position for a node
// rather than the node itself (such as InsertedSemicolon). Empty nodes do not
// become parents of other empty nodes.
//
// Note: errors reported via ShouldTryToRecover might be reported out of order
// with ParserListener, but SyntaxProblem nodes produced by error recovery will
// be delivered here as usual syntactic constructs.
type ParserListener func(t node.Type, offset, endoffset int)

// HasParser returns whether the language has a registered parser.
func HasParser(lang lpb.Language) bool {
	_, ok := parsers[lang]
	return ok
}

// Parse uses a parser for the given language to report all syntactic elements
// from "source". It is the responsibility of the caller to decide which nodes
// need to be preserved and build an AST. The function returns true if the
// parsing succeeds, i.e. there were no syntax errors, or the parser was able
// to recover from all of them. Broken code is reported as nodes of the
// SyntaxProblem category.
//
// The provided error handler is called for all positions where the parser
// stumbled upon a syntax error or skipped an unrecognized token.
//
// Note: parsing is slower than lexing but one can expect at least 10MB/s
// of throughput (and usually no memory allocations) from this function.
func Parse(ctx context.Context, lang lpb.Language, source string, pl ParserListener, opts Options) error {
	if parser, ok := parsers[lang]; ok {
		return parser(ctx, source, pl, opts)
	}
	return ErrUnsupportedLanguage
}

// Parser is an actual implementation of the parser for some language.
type Parser func(ctx context.Context, source string, l ParserListener, opts Options) error

// Lexer is an actual implementation of the lexer for some language.
type Lexer func(ctx context.Context, source string, l LexerListener)

var lexers = map[lpb.Language]Lexer{}
var parsers = map[lpb.Language]Parser{}

// RegisterLexer adds the lexer implementation to the registry.
func RegisterLexer(l lpb.Language, lexer Lexer) {
	lexers[l] = lexer
}

// RegisterParser adds the parser implementation to the registry.
func RegisterParser(l lpb.Language, parser Parser) {
	parsers[l] = parser
}
