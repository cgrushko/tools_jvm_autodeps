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

package java

import (
	lpb "github.com/bazelbuild/tools_jvm_autodeps/thirdparty/golang/parsers/lang"
	"github.com/bazelbuild/tools_jvm_autodeps/thirdparty/golang/parsers/node"
	"github.com/bazelbuild/tools_jvm_autodeps/thirdparty/golang/parsers/parsers"
	tpb "github.com/bazelbuild/tools_jvm_autodeps/thirdparty/golang/parsers/public/token"
	"context"
)

const (
	keywordStart     = ABSTRACT
	keywordEnd       = WITH + 1
	punctuationStart = LPAREN
	punctuationEnd   = GTGTGTASSIGN + 1
)

func lex(ctx context.Context, source string, listener parsers.LexerListener) {
	l := new(Lexer)
	l.Init(source)
	for tok := l.Next(); tok != EOI; tok = l.Next() {
		s, e := l.Pos()

		switch {
		case tok >= keywordStart && tok < keywordEnd:
			listener(tpb.TokenType_KEYWORD, s, e)
			continue
		case tok >= punctuationStart && tok < punctuationEnd:
			listener(tpb.TokenType_PUNCTUATION, s, e)
			continue
		}
		switch tok {
		case IDENTIFIER:
			listener(tpb.TokenType_IDENTIFIER, s, e)
		case TRADITIONALCOMMENT, ENDOFLINECOMMENT:
			listener(tpb.TokenType_COMMENT, s, e)
		case INTEGERLITERAL, FLOATINGPOINTLITERAL:
			listener(tpb.TokenType_NUMERIC_LITERAL, s, e)
		case CHARACTERLITERAL, STRINGLITERAL:
			listener(tpb.TokenType_STRING_LITERAL, s, e)
		case INVALID_TOKEN:
			listener(tpb.TokenType_ERROR_TOKEN, s, e)
		}
	}
}

func parse(ctx context.Context, source string, listener parsers.ParserListener, opts parsers.Options) error {
	l := new(Lexer)
	l.Init(source)
	p := new(Parser)
	p.Init(Listener(listener))
	p.IncludeAllTokens = opts.IncludeAllTokens
	if err := p.Parse(ctx, l); err != nil {
		if se, ok := err.(SyntaxError); ok {
			return parsers.SyntaxError{
				Description: "syntax error",
				Line:        se.Line,
				Offset:      se.Offset,
				Length:      se.Endoffset - se.Offset,
			}
		}
		return err
	}
	listener(node.JavaFile, 0, len(source))
	return nil
}

func init() {
	parsers.RegisterLexer(lpb.Language_JAVA, lex)
	parsers.RegisterParser(lpb.Language_JAVA, parse)
	if err := node.RegisterTypes(lpb.Language_JAVA, AllTypes[:]); err != nil {
		panic(err)
	}
}
