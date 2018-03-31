# Copyright 2018 The Jadep Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Java SE 9 Edition.
#
# based on:
#   https://docs.oracle.com/javase/specs/jls/se9/html/jls-3.html (lexical structure)
#   https://docs.oracle.com/javase/specs/jls/se9/html/jls-19.html (syntax)
#
# Also see http://llbit.se/?p=2217 for a nice summary of parsing problems
# in Java 8.

language java(go);

lang = "java"
package = "jadep/thirdparty/golang/parsers/java"
eventBased = true
cancellable = true
reportTokens = [TraditionalComment, EndOfLineComment, Identifier, invalid_token]
extraTypes = ["File"]

:: lexer

# 3.5. Input Elements and Tokens

eoi: /\x1a/

# 3.6 White Space

WhiteSpace: /[\r\n\t\f\x20]+/    (space)

# 3.7 Comments

commentContent = /([^*]|\*+[^\/*])*\**/
TraditionalComment: /\/\*{commentContent}\*\// (space)

# Disable backtracking for incomplete /* */ comments:
invalid_token: /\/\*{commentContent}/

EndOfLineComment: /\/\/[^\r\n]*/ (space)

# 3.8 Identifiers

Identifier: /{JavaLetter}{JavaLetterOrDigit}*/  (class)

# Identifiers with incomplete escape sequences.
invalid_token: /({JavaLetter}{JavaLetterOrDigit}*)?\\(u+{HexDigit}{0,3})?/

JavaLetter = /[a-zA-Z_$\p{L}\p{Nl}]|{UnicodeEscape}/
JavaLetterOrDigit = /[a-zA-Z0-9_$\p{L}\p{Nl}\p{Nd}]|{UnicodeEscape}/

# 3.9 Keywords

# Note: a single underscore is a reserved keyword since Java 9.

'abstract': /abstract/
'assert': /assert/
'boolean': /boolean/
'break': /break/
'byte': /byte/
'case': /case/
'catch': /catch/
'char': /char/
'class': /class/  { l.ordinaryUnit = true }
'const': /const/  # reserved, unused
'continue': /continue/
'default': /default/
'do': /do/
'double': /double/
'else': /else/
'enum': /enum/  { l.ordinaryUnit = true }
'extends': /extends/
'false': /false/
'final': /final/
'finally': /finally/
'float': /float/
'for': /for/
'goto': /goto/  # reserved, unused
'if': /if/
'implements': /implements/
'import': /import/
'instanceof': /instanceof/
'int': /int/
'interface': /interface/  { l.ordinaryUnit = true }
'long': /long/
'native': /native/
'new': /new/
'null': /null/
'package': /package/
'private': /private/
'protected': /protected/
'public': /public/
'return': /return/
'short': /short/
'static': /static/
'strictfp': /strictfp/
'super': /super/
'switch': /switch/
'synchronized': /synchronized/
'this': /this/
'throw': /throw/
'throws': /throws/
'transient': /transient/
'true': /true/
'try': /try/
'void': /void/
'volatile': /volatile/
'while': /while/

# (spec) The following character sequences are tokenized as keywords solely
# where they appear as terminals in the ModuleDeclaration and ModuleDirective
# productions.

# not after '@', 'package', 'import', 'static', or '.'. Also, not in ().
'open': /open/  { if l.moduleAsID() { token = IDENTIFIER } }
'module': /module/  { if l.moduleAsID() { token = IDENTIFIER } else { l.moduleUnit = true } }

# after '{' or ';' only
'requires': /requires/  { if l.directiveAsID() { token = IDENTIFIER } }
'exports': /exports/  { if l.directiveAsID() { token = IDENTIFIER } }
'opens': /opens/  { if l.directiveAsID() { token = IDENTIFIER } }
'uses': /uses/  { if l.directiveAsID() { token = IDENTIFIER } }
'provides': /provides/  { if l.directiveAsID() { token = IDENTIFIER } }

# after 'requires' and 'static' only
'transitive': /transitive/ { if !l.moduleUnit || l.token != REQUIRES && l.token != STATIC { token = IDENTIFIER } }

# after identifiers only
'to': /to/  { if !l.moduleUnit || l.token != IDENTIFIER { token = IDENTIFIER } }
'with': /with/  { if !l.moduleUnit || l.token != IDENTIFIER { token = IDENTIFIER } }

# 3.10.1 Integer Literals

IntegerLiteral: /(0|[1-9](_*{Digits})?)[lL]?/
IntegerLiteral: /{HexNumeral}[lL]?/
IntegerLiteral: /0_*{OctalDigits}[lL]?/
IntegerLiteral: /0[bB]{BinaryDigits}[lL]?/

Digits = /{Digit}(_*{Digit})*/
Digit = /[0-9]/

HexNumeral = /0[xX]{HexDigits}/
HexDigits = /{HexDigit}(_*{HexDigit})*/
HexDigit = /[0-9a-fA-F]/

OctalDigits = /{OctalDigit}(_*{OctalDigit})*/
OctalDigit = /[0-7]/

BinaryDigits = /[01](_*[01])*/

invalid_token: /0[bBxX]/
invalid_token: /{Digits}_*/ -1  /* This covers both octal and decimal literals */
invalid_token: /{HexNumeral}_+/
invalid_token: /0[bB]{BinaryDigits}_+/

# 3.10.2. Floating-Point Literals

FloatingPointLiteral: /({Digits}\.{Digits}?|\.{Digits}){ExponentPart}?{FloatTypeSuffix}?/
FloatingPointLiteral: /{Digits}{ExponentPart}{FloatTypeSuffix}?/
FloatingPointLiteral: /{Digits}{FloatTypeSuffix}/

ExponentPart = /[eE][+-]?{Digits}/
BrokenExponent = /[eE][+-]?({Digits}_+)?/
FloatTypeSuffix = /[fFdD]/

invalid_token: /{Digits}\.?{BrokenExponent}/
invalid_token: /{Digits}?\.{Digits}_+/
invalid_token: /{Digits}?\.{Digits}{BrokenExponent}/

FloatingPointLiteral: /{HexSignificand}{BinaryExponent}{FloatTypeSuffix}?/
BinaryExponent = /[pP][+-]?{Digits}/
BrokenBinaryExponent = /[pP][+-]?({Digits}_+)?/

HexSignificand = /{HexNumeral}\.?|0[xX]{HexDigits}?\.{HexDigits}/

invalid_token: /0[xX]{HexDigits}?\.({HexDigits}_*)?/
invalid_token: /{HexSignificand}{BrokenBinaryExponent}/

# 3.10.3. Boolean Literals (see in the keywords section)

# 3.10.4-6 String Literals

EscapeSequence = /\\[btnfr"'\\]|{OctalEscape}/
OctalEscape = /\\([0-3]?[0-7])?[0-7]/

CharacterLiteral: /'([^\r\n'\\]|{EscapeSequence}|{UnicodeEscape})'/

StringLiteral: /"([^\r\n"\\]|{EscapeSequence}|{UnicodeEscape})*"/

UnicodeEscape = /\\u+{HexDigit}{4}/

# 3.10.7. The Null Literal (see in the keywords section)

# 3.11. Separators

'(': /\(/  { l.depth++ }
')': /\)/  { l.depth-- }
'{': /{/
'}': /}/
'[': /\[/
']': /\]/
';': /;/
',': /,/
'.': /\./
invalid_token: /\.\./
'...': /\.\.\./
'@': /@/
'::': /::/

# 3.12. Operators

'=': /=/
'>': />/
'<': /</
'!': /!/
'~': /~/
'?': /?/
':': /:/
'->': /->/
'==': /==/
'>=': />=/
'<=': /<=/
'!=': /!=/
'&&': /&&/
'||': /\|\|/
'++': /\+\+/
'--': /--/
'+': /+/
'-': /-/
'*': /\*/
'/': /\//
'&': /&/
'|': /\|/
'^': /\^/
'%': /%/
'<<': /<</
'>>': />>/
'>>>': />>>/
'+=': /+=/
'-=': /-=/
'*=': /\*=/
'/=': /\/=/
'&=': /&=/
'|=': /\|=/
'^=': /^=/
'%=': /%=/
'<<=': /<<=/
'>>=': />>=/
'>>>=': />>>=/

invalid_token:

:: parser

%input CompilationUnit;

# We use lookaheads to disambiguate casts, lambdas, and parenthesized
# expressions.
%assert empty set(first AfterReferenceTypeCast & follow Primary);

# NoName excludes QualifiedName from each production that accepts it as an
# argument.
%flag NoName = false;

# In switch statements, default can mean both - a start of a local variable
# declaration and the default case, so we exclude 'default' from the list of
# valid modifiers everywhere except in interface declarations.
%flag WithDefault = false;

IdentifierName -> IdentifierName :
    Identifier ;

%interface Modifier;

Modifier<WithDefault> -> Modifier :
    Annotation
  | ModifierKeyword
;

ModifierKeyword<WithDefault> -> ModifierKeyword :
    'public'
  | 'protected'
  | 'private'
  | 'abstract'                  # interfaces, annotation types
  | 'static'                    # interfaces, constants
  | 'final'                     # local variables, parameters, constants
  | 'strictfp'                  # interfaces
  | 'transient'
  | 'volatile'
  | 'synchronized'              # methods
  | 'native'                    # methods
  | [WithDefault] 'default'     # interface methods
;

Modifiers<WithDefault> :
    Modifier
  | Modifiers Modifier
;

# 3. Lexical Structure

Literal -> Literal :
    IntegerLiteral
  | FloatingPointLiteral
  | CharacterLiteral
  | StringLiteral
  | 'true'
  | 'false'
  | 'null'
;

# 4. Types, Values, and Variables

# Modifiers are usually shared between the type and the construction which
# starts with the type, so a common pattern is to replace Type with:
#   Modifiers? Type<+NoModifiers>.
# This also solves a bunch of shift/reduce conflicts.
%flag NoModifiers = false;

%interface Type;

Type<NoModifiers> -> Type :
    PrimitiveType
  | ReferenceType
  | 'void'                                                 -> VoidType
;

PrimitiveType<NoModifiers> -> PrimitiveType :
    NumericType
  | 'boolean'
  | [!NoModifiers] Modifiers NumericType
  | [!NoModifiers] Modifiers 'boolean'
;

NumericType :
    'byte'
  | 'short'
  | 'int'
  | 'long'
  | 'char'
  | 'float'
  | 'double'
;

ReferenceType<NoModifiers, NoName> -> Type :
    ClassType<+NoName>
  | [!NoName] (QualifiedName -> TypeName)                  -> ClassType
  | ArrayType
;

# This is a class type without starting Modifiers and ending TypeArguments.
ClassRefNoName :
    ((QualifiedName -> TypeName) TypeArguments -> ClassType) '.' Annotation? Identifier
  | ((QualifiedName -> TypeName) -> ClassType) '.' Annotation Identifier
  | (ClassRefNoName TypeArguments? -> ClassType) '.' Annotation? Identifier
;

# Class and reference types can often be found in positions where simple
# qualified names can start something else, so we postpone the decision whether
# a qualified name is a type as much as we can.
#   f(java.lang.Runnable::run)   // method reference starting with a type
#   f(a.b.c)                     // variable
ClassType<NoModifiers, NoName> -> ClassType :
    [!NoName] (QualifiedName -> TypeName)  -> ClassType
  | (QualifiedName -> TypeName) TypeArguments
  | ClassRefNoName TypeArguments?
  | [!NoModifiers] Modifiers (ClassRefNoName | QualifiedName -> TypeName) TypeArguments?
;

# This is ClassType + '>', expanded to cover >> and >>>.
# Note: as a postprocessing step we substract one from its 'endoffset' and make it ClassType.
ClassType1 :
    ClassType '>'
  | Modifiers? (ClassRefNoName | QualifiedName -> TypeName) '<' TypeArgumentList '>>'
          {
            p.listener(node.JavaClassType, lhs.sym.offset, lhs.sym.endoffset - 1);
          }
  | Modifiers? (ClassRefNoName | QualifiedName -> TypeName) '<' (TypeArgumentList ',')? UnclosedTypeArgument '>>>'
          {
            p.listener(node.JavaClassType, p.lastClassRefOffset, lhs.sym.endoffset - 2);
            p.listener(node.JavaTypeArgument, ${UnclosedTypeArgument.offset}, lhs.sym.endoffset - 2);
            p.listener(node.JavaClassType, lhs.sym.offset, lhs.sym.endoffset - 1);
          }
;

ArrayType<NoModifiers> -> ArrayType :
    PrimitiveType Dims
  | ClassType<+NoName> Dims
  | ((QualifiedName -> TypeName) -> ClassType) Dims
;

Dims :
    Annotations? '[' ']'       -> Dim
  | Dims (Annotations? '[' ']' -> Dim)
;

TypeParameter -> TypeParameter :
    Annotations? IdentifierName TypeBound? ;

TypeBound -> TypeBound :
    'extends' ClassType AdditionalBound+? ;

AdditionalBound :
    '&' ClassType ;

# This also covers the diamond operator.
TypeArguments -> TypeArguments :
    '<' TypeArgumentList? '>'
  | '<' (TypeArgumentList ',')? UnclosedTypeArgument '>>'
          {
            p.listener(node.JavaClassType, p.lastClassRefOffset, lhs.sym.endoffset - 1);
            p.listener(node.JavaTypeArgument, ${UnclosedTypeArgument.offset}, lhs.sym.endoffset - 1);
          }
  | '<' (TypeArgumentList ',')? UnclosedTypeArgument2 '>>>'
          {
            p.listener(node.JavaClassType, p.lastClassRefOffset, lhs.sym.endoffset - 2);
            p.listener(node.JavaTypeArgument, p.lastTypeArgOffset, lhs.sym.endoffset - 2);
            p.listener(node.JavaClassType, p.lastClassRefOffset2, lhs.sym.endoffset - 1);
            p.listener(node.JavaTypeArgument, ${UnclosedTypeArgument2.offset}, lhs.sym.endoffset - 1);
          }
;

TypeArgumentList :
    TypeArgument
  | TypeArgumentList ',' TypeArgument
;

TypeArgument -> TypeArgument :
    Modifiers? ReferenceType<+NoModifiers>
  | Modifiers? '?' (BoundType ReferenceType)?
;

# This is TypeArgument without the last closing '>'.
# Note: as a postprocessing step we add one > from the next token to the
# covered range and make it TypeArgument.
UnclosedTypeArgument :
    (Modifiers? '?' BoundType)? Modifiers? ClassRefNoName '<' TypeArgumentList
        {
          p.lastClassRefOffset = ${ClassRefNoName.offset};
        }
  | (Modifiers? '?' BoundType)? Modifiers? (QualifiedName -> TypeName) '<' TypeArgumentList
        {
          p.lastClassRefOffset = ${QualifiedName.offset};
        }
;

# This is TypeArgument without the two last closing '>'.
# Note: as a postprocessing step we add two >> from the next token to the
# covered range and make it TypeArgument.
UnclosedTypeArgument2 :
    (Modifiers? '?' BoundType)? Modifiers? ClassRefNoName '<' (TypeArgumentList ',')? UnclosedTypeArgument
        {
          p.lastClassRefOffset2 = ${ClassRefNoName.offset};
          p.lastTypeArgOffset = ${UnclosedTypeArgument.offset};
        }
  | (Modifiers? '?' BoundType)? Modifiers? (QualifiedName -> TypeName) '<' (TypeArgumentList ',')? UnclosedTypeArgument
        {
          p.lastClassRefOffset2 = ${QualifiedName.offset};
          p.lastTypeArgOffset = ${UnclosedTypeArgument.offset};
        }
;

BoundType -> BoundType :
    'extends'
  | 'super'
;

# 6. Names

QualifiedName :
    Identifier
  | QualifiedName '.' Identifier
;

MethodName -> MethodName :
    Identifier ;

# 7. Packages and Modules

CompilationUnit :
    PackageDeclaration? ImportDeclaration+? TypeDeclaration+?
  | ImportDeclaration+? ModuleDeclaration
;

PackageDeclaration -> Package :
    Modifiers? 'package' Name ';' ;

Name -> Name :
    identifiers=QualifiedName ;

ImportName :
    identifiers+=Identifier                                -> Name
  | identifiers=QualifiedName '.' identifiers+=Identifier  -> Name
  | identifiers=QualifiedName '.' '*'                      -> NameStar
;

ImportDeclaration -> Import :
    'import' Static? ImportName ';' ;

Static -> Static :
    'static' ;

%interface TypeDeclaration;

TypeDeclaration -> TypeDeclaration :
    ClassDeclaration
  | InterfaceDeclaration
  | ';'                                   -> EmptyDecl
;

ModuleDeclaration -> ModuleDeclaration:
    Modifiers? 'open'? 'module' Name '{' ModuleDirective* '}' ;

ModuleName -> ModuleName :
    QualifiedName ;

PackageName -> PackageName :
    QualifiedName ;

ModuleDirective -> ModuleDirective:
    'requires' RequiresModifier+? ModuleName ';'
  | 'exports' PackageName ('to' (ModuleName separator ',')+)? ';'
  | 'opens' PackageName ('to' (ModuleName separator ',')+)? ';'
  | 'uses' TypeName ';'
  | 'provides' TypeName 'with' (TypeName separator ',')+ ';'
;

RequiresModifier -> ModifierKeyword:
    'transitive' | 'static' ;

# 8. Classes

%interface ClassDeclaration;

ClassDeclaration<WithDefault> -> ClassDeclaration :
    NormalClassDeclaration
  | EnumDeclaration
;

NormalClassDeclaration<WithDefault> -> Class :
    Modifiers? 'class' IdentifierName TypeParameters? Superclass? Superinterfaces? ClassBody
;

TypeParameters -> TypeParameters :
    '<' (TypeParameterList ',')? TypeParameter1 ;

TypeParameter1 :
    Annotations? IdentifierName '>'
          {
            p.listener(node.JavaTypeParameter, lhs.sym.offset, lhs.sym.endoffset - 1);
          }
  | Annotations? IdentifierName extends='extends' (ClassType AdditionalBound+? '&')? ClassType1
          {
            p.listener(node.JavaTypeBound, ${extends.offset}, lhs.sym.endoffset - 1);
            p.listener(node.JavaTypeParameter, lhs.sym.offset, lhs.sym.endoffset - 1);
          }
;

TypeParameterList :
    TypeParameter
  | TypeParameterList ',' TypeParameter
;

Superclass -> Extends :
    'extends' ClassType ;

Superinterfaces -> Implements :
    'implements' (ClassType separator ',')+ ;

ClassBody -> Body :
    '{' ClassBodyDeclaration* '}' ;

%interface MemberDeclaration;

ClassBodyDeclaration -> MemberDeclaration :
    ClassMemberDeclaration
  | InstanceInitializer
  | StaticInitializer
  | ConstructorDeclaration
;

ClassMemberDeclaration -> MemberDeclaration :
    FieldDeclaration
  | MethodDeclaration
  | ClassDeclaration
  | InterfaceDeclaration
  | ';'                                                    -> EmptyDecl
;

FieldDeclaration<WithDefault> -> Field :
    Modifiers? Type<+NoModifiers> VariableDeclaratorList ';' ;

VariableDeclaratorList :
    VariableDeclarator
  | VariableDeclaratorList ',' VariableDeclarator
;

VariableDeclarator -> VarDecl :
    VariableDeclaratorId ('=' VariableInitializer)? ;

VariableDeclaratorId :
    IdentifierName Dims? ;

%interface Initializer;

VariableInitializer -> Initializer:
    Expression                                             -> InitializerExpression
  | ArrayInitializer
;

MethodDeclaration<WithDefault> -> Method :
    Modifiers? MethodHeader MethodBody ;

MethodHeader :
    (TypeParameters Annotations?)? Type<+NoModifiers> MethodDeclarator Throws? ;

MethodDeclarator :
    IdentifierName FormalParameters Dims? ;

FormalParameters -> FormalParameters :
    '(' FormalParameterList? ')' ;

FormalParameterList :
    Parameter
  | FormalParameterList ',' Parameter
;

%interface Parameter;

# TODO: Replace (ClassType|PrimitiveType) Dims? with Type
Parameter -> Parameter :
    Modifiers? (ClassType<+NoModifiers> | PrimitiveType<+NoModifiers>)
        Dims? (Annotations? '...')? VariableDeclaratorId   -> FormalParameter
  | Modifiers? (ClassType<+NoModifiers> | PrimitiveType<+NoModifiers>)
        Dims? (Identifier '.')? ('this' -> IdentifierName) -> ReceiverParameter
;

Throws -> Throws :
    'throws' (ClassType separator ',')+  ;

%interface MethodBody;

MethodBody -> MethodBody:
    Block
  | ';'                                                    -> NoBody
;

InstanceInitializer -> InstanceInitializer :
    Block ;

StaticInitializer -> StaticInitializer :
    'static' Block ;

ConstructorDeclaration -> Constructor :
    Modifiers? ConstructorDeclarator Throws? ConstructorBody ;

ConstructorDeclarator :
    TypeParameters? IdentifierName FormalParameters ;

ConstructorBody -> Block :
    '{' ExplicitConstructorInvocation? BlockStatements? '}' ;

%interface ConstructorInvocation;

ExplicitConstructorInvocation -> ConstructorInvocation :
    TypeArguments? 'this' '(' ArgumentList? ')' ';'                       -> ThisCall
  | TypeArguments? 'super' '(' ArgumentList? ')' ';'                      -> SuperCall
  | (QualifiedName -> ExprName) '.' TypeArguments? 'super' '(' ArgumentList? ')' ';'    -> SuperCall
  | Primary '.' TypeArguments? 'super' '(' ArgumentList? ')' ';'          -> SuperCall
;

EnumDeclaration<WithDefault> -> Enum :
    Modifiers? 'enum' IdentifierName Superinterfaces? EnumBody ;

# TODO: rename into EnumBody?
EnumBody -> Body :
    '{' EnumConstantList? ','? EnumBodyDeclarations? '}' ;

EnumConstantList :
    EnumConstant
  | EnumConstantList ',' EnumConstant
;

EnumConstant -> EnumConstant :
    Modifiers? IdentifierName ('(' ArgumentList? ')')? ClassBody? ;

EnumBodyDeclarations :
    ';' ClassBodyDeclaration* ;

# 9. Interfaces

%interface InterfaceDeclaration;

InterfaceDeclaration<WithDefault> -> InterfaceDeclaration :
    NormalInterfaceDeclaration
  | AnnotationTypeDeclaration
;

NormalInterfaceDeclaration<WithDefault> -> Interface :
    Modifiers? 'interface' IdentifierName TypeParameters? ExtendsInterfaces? InterfaceBody ;

ExtendsInterfaces -> Extends :
    'extends' (ClassType separator ',')+ ;

InterfaceBody -> Body :
    '{' InterfaceMemberDeclaration<+WithDefault>* '}' ;

InterfaceMemberDeclaration<WithDefault> -> MemberDeclaration :
    FieldDeclaration
  | MethodDeclaration
  | ClassDeclaration
  | InterfaceDeclaration
  | ';'                                                    -> EmptyDecl
;

AnnotationTypeDeclaration<WithDefault> -> AnnotationType :
    Modifiers? '@' 'interface' IdentifierName AnnotationTypeBody ;

AnnotationTypeBody -> Body :
    '{' AnnotationTypeMemberDeclaration* '}' ;

AnnotationTypeMemberDeclaration -> MemberDeclaration :
    AnnotationTypeElementDeclaration
  | FieldDeclaration
  | ClassDeclaration
  | InterfaceDeclaration
  | ';'                                                    -> EmptyDecl
;

AnnotationTypeElementDeclaration -> AnnotationTypeElement :
    Modifiers? Type<+NoModifiers> IdentifierName '(' ')' Dims? DefaultValue? ';' ;

DefaultValue -> DefaultValue :
    'default' ElementValue ;

TypeName -> TypeName :
    QualifiedName ;

Annotation -> Annotation :
    '@' TypeName
  | '@' TypeName '(' ElementValue ')'
  | '@' TypeName '(' ElementValuePairList? ')'
;

Annotations :
    Annotation
  | Annotations Annotation
;

ElementValuePairList :
    ElementValuePair
  | ElementValuePairList ',' ElementValuePair
;

ElementValuePair -> ElementValuePair :
    Identifier '=' ElementValue ;

# TODO: make interface?
ElementValue -> ElementValue :
    ConditionalExpression
  | ElementValueArrayInitializer
  | Annotation
;

ElementValueArrayInitializer -> ArrayInitializer :
    '{' ElementValueList? ','? '}' ;

ElementValueList :
    ElementValue
  | ElementValueList ',' ElementValue
;

# 10. Arrays

ArrayInitializer -> ArrayInitializer :
    '{' VariableInitializerList? ','? '}' ;

VariableInitializerList :
    VariableInitializer
  | VariableInitializerList ',' VariableInitializer
;

# 14. Blocks and Statements

%interface Statement;

Block -> Block :
    '{' BlockStatements? '}' ;

BlockStatements :
    BlockStatement
  | BlockStatements BlockStatement
;

BlockStatement -> Statement :
    LocalVariableDeclarationStatement
  | ClassDeclaration
  | Statement
;

LocalVariableDeclarationStatement -> LocalVars :
    LocalVariableDeclaration ';' ;

LocalVariableDeclaration :
    Modifiers? Type<+NoModifiers> VariableDeclaratorList ;

Statement -> Statement :
    StatementWithoutTrailingSubstatement
  | LabeledStatement
  | IfThenStatement
  | IfThenElseStatement
  | WhileStatement
  | ForStatement
;

StatementNoShortIf -> Statement :
    StatementWithoutTrailingSubstatement
  | LabeledStatementNoShortIf
  | IfThenElseStatementNoShortIf
  | WhileStatementNoShortIf
  | ForStatementNoShortIf
;

StatementWithoutTrailingSubstatement -> Statement :
    Block
  | EmptyStatement
  | ExpressionStatement
  | AssertStatement
  | SwitchStatement
  | DoStatement
  | BreakStatement
  | ContinueStatement
  | ReturnStatement
  | SynchronizedStatement
  | ThrowStatement
  | TryStatement
;

EmptyStatement -> EmptyStatement :
    ';' ;

LabeledStatement -> Labeled :
    IdentifierName ':' Statement ;

LabeledStatementNoShortIf -> Labeled :
    IdentifierName ':' StatementNoShortIf ;

ExpressionStatement -> ExpressionStatement :
    StatementExpression ';' ;

StatementExpression -> Expression :
    Assignment
  | PreIncrementExpression
  | PreDecrementExpression
  | PostIncrementExpression
  | PostDecrementExpression
  | MethodInvocation
  | ClassInstanceCreationExpression
;

IfThenStatement -> If :
    'if' '(' Expression ')' then=Statement ;

IfThenElseStatement -> If :
    'if' '(' Expression ')' then=StatementNoShortIf 'else' else=Statement ;

IfThenElseStatementNoShortIf -> If :
    'if' '(' Expression ')' then=StatementNoShortIf 'else' else=StatementNoShortIf ;

AssertStatement -> Assert :
    'assert' expr=Expression ';'
  | 'assert' expr=Expression ':' message=Expression ';'
;

SwitchStatement -> Switch :
    'switch' '(' Expression ')' SwitchBlock ;

SwitchBlock -> SwitchBlock :
    '{' SwitchItem+? '}' ;

%interface SwitchItem;

SwitchItem -> SwitchItem :
    BlockStatement
  | SwitchLabel
;

SwitchLabel -> SwitchItem :
    'case' Expression ':'                                  -> Case
  | 'default' ':'                                          -> DefaultCase
;

WhileStatement -> While :
    'while' '(' Expression ')' Statement
;

WhileStatementNoShortIf -> While :
    'while' '(' Expression ')' StatementNoShortIf
;

DoStatement -> DoWhile :
    'do' Statement 'while' '(' Expression ')' ';' ;

ForStatement -> Statement :
    BasicForStatement
  | EnhancedForStatement
;

ForStatementNoShortIf -> Statement :
    BasicForStatementNoShortIf
  | EnhancedForStatementNoShortIf
;

BasicForStatement -> BasicFor :
    'for' '(' ForInit? ';' Expression? ';' ForUpdate? ')' Statement ;

BasicForStatementNoShortIf -> BasicFor :
    'for' '(' ForInit? ';' Expression? ';' ForUpdate? ')' StatementNoShortIf ;

ForInit -> ForInit :
    StatementExpressionList
  | LocalVariableDeclaration
;

ForUpdate -> ForUpdate :
    StatementExpressionList ;

StatementExpressionList :
    StatementExpression
  | StatementExpressionList ',' StatementExpression
;

EnhancedForStatement -> EnhFor :
    'for' '(' Modifiers? Type<+NoModifiers> VariableDeclaratorId ':' Expression ')' Statement ;

EnhancedForStatementNoShortIf -> EnhFor :
    'for' '(' Modifiers? Type<+NoModifiers> VariableDeclaratorId ':' Expression ')' StatementNoShortIf ;

BreakStatement -> Break :
    'break' Identifier? ';' ;

ContinueStatement -> Continue :
    'continue' Identifier? ';' ;

ReturnStatement -> Return :
    'return' Expression? ';' ;

ThrowStatement -> Throw :
    'throw' Expression ';' ;

SynchronizedStatement -> Synchronized :
    'synchronized' '(' Expression ')' Block ;

# TODO: just Try?
TryStatement -> TryStatement :
    'try' Block Catches
  | 'try' Block Catches? Finally
  | 'try' ResourceSpecification Block Catches? Finally?
;

Catches :
    CatchClause
  | Catches CatchClause
;

CatchClause -> Catch :
    'catch' '(' CatchFormalParameter ')' Block ;

CatchFormalParameter -> CatchParameter :
    Modifiers? CatchType VariableDeclaratorId ;

CatchType :
    ClassType<+NoModifiers>
  | CatchType '|' ClassType ;

Finally -> Finally :
    'finally' Block ;

ResourceSpecification -> ResourceSpecification :
    '(' ResourceList ';'? ')' ;

ResourceList :
    Resource
  | ResourceList ';' Resource
;

Resource -> Resource :
    Modifiers? Type<+NoModifiers> VariableDeclaratorId '=' Expression
  | VariableAccess
;

VariableAccess:
    QualifiedName    -> ExprName
  | FieldAccess
;

# 15. Expressions

%interface Expression;

Primary -> Expression :
    PrimaryNoNewArray
  | ArrayCreationExpression
;

PrimaryNoNewArray -> Expression :
    Literal
  | ClassLiteral
  | 'this'                                                                 -> This
  | (QualifiedName -> TypeName) '.' 'this'                                 -> This
  | '(' (?= !CastStartLookahead & !LambdaStartLookahead) Expression ')'    -> Parenthesized
  | ClassInstanceCreationExpression
  | FieldAccess
  | ArrayAccess
  | MethodInvocation
;

ClassLiteral -> ClassLiteral :
    (QualifiedName -> TypeName) Dims? '.' 'class'
  | PrimitiveType<+NoModifiers> Dims? '.' 'class'
  | 'void' '.' 'class'
;

ClassInstanceCreationExpression -> Expression :
    UnqualifiedClassInstanceCreationExpression
  | (QualifiedName -> ExprName) '.' UnqualifiedClassInstanceCreationExpression  -> QualifiedNew
  | Primary '.' UnqualifiedClassInstanceCreationExpression                      -> QualifiedNew
;

UnqualifiedClassInstanceCreationExpression -> New :
    'new' TypeArguments? ClassType '(' ArgumentList? ')' ClassBody?
;

SuperRef -> SuperRef :
    'super'
  | (QualifiedName -> TypeName) '.' 'super'
;

FieldAccess -> FieldAccess :
    Primary '.' Identifier
  | SuperRef '.' Identifier
;

ArrayAccess -> ArrayAccess :
    (QualifiedName -> ExprName) '[' Expression ']'
  | PrimaryNoNewArray '[' Expression ']'
;

MethodInvocation -> MethodInvocation :
    MethodName '(' ArgumentList? ')'
  | (QualifiedName -> TypeOrExprName) '.' TypeArguments? MethodName '(' ArgumentList? ')'
  | Primary '.' TypeArguments? MethodName '(' ArgumentList? ')'
  | SuperRef '.' TypeArguments? MethodName '(' ArgumentList? ')'
;

ArgumentList :
    Expression
  | ArgumentList ',' Expression
;

MethodReference -> MethodReference :
    Primary '::' TypeArguments? Identifier
  | SuperRef '::' TypeArguments? Identifier
  | Modifiers? MethodReferenceType '::' TypeArguments? ('new' | Identifier)
;

MethodReferenceLookahead :
    TypeArguments ('.' Annotation? Identifier TypeArguments?)+? Dims? '::' ;

MethodReferenceType :
    PrimitiveType<+NoModifiers> Dims                       -> ArrayType
  | ClassType2
  | ClassType2 Dims                                        -> ArrayType
  | QualifiedName                                          -> TypeOrExprName
  | ((QualifiedName -> TypeName) -> ClassType) Dims        -> ArrayType
;

# This is a class type, but with a lookahead token before the first
# TypeArguments.
ClassType2 -> ClassType :
    (QualifiedName -> TypeName) (?= MethodReferenceLookahead) TypeArguments
  | ((QualifiedName -> TypeName) -> ClassType) '.' Annotation Identifier TypeArguments?
  | ClassType2 '.' Annotation? Identifier TypeArguments?
;

ArrayCreationExpression -> NewArray :
    'new' PrimitiveType DimExprs Dims?
  | 'new' ClassType DimExprs Dims?
  | 'new' PrimitiveType Dims ArrayInitializer
  | 'new' ClassType Dims ArrayInitializer
;

DimExprs :
    DimExpr
  | DimExprs DimExpr
;

DimExpr -> DimExpr :
    Annotations? '[' Expression ']' ;

Expression -> Expression :
    LambdaOrMethodReference
  | ConditionalExpression
  | Assignment
  | '(' (?= CastStartLookahead & !LambdaStartLookahead) ReferenceType AdditionalBound+? ')'
      LambdaOrMethodReference   -> CastExpression
;

LambdaOrMethodReference -> Expression :
    MethodReference
  | LambdaExpression
;

LambdaStartLookahead :
    FormalParameterList? ')' '->'
  | Identifier (',' Identifier)+? ')' '->'
;

LambdaExpression -> Lambda :
    LambdaParameters '->' LambdaBody ;

LambdaParameters -> LambdaParameters :
    (Identifier -> IdentifierName)
  | '(' (?= LambdaStartLookahead) FormalParameterList? ')'
  | '(' (?= LambdaStartLookahead) IdentifierName (',' IdentifierName)+? ')'
;

LambdaBody :
    Expression
  | Block
;

Assignment -> Expression :
    LeftHandSide AssignmentOp Expression -> Assignment ;

LeftHandSide :
    QualifiedName    -> ExprName
  | FieldAccess
  | ArrayAccess
;

AssignmentOp -> AssignmentOp :
    '='
  | '*='
  | '/='
  | '%='
  | '+='
  | '-='
  | '<<='
  | '>>='
  | '>>>='
  | '&='
  | '^='
  | '|='
;

ConditionalExpression -> Expression :
    LogicalExpression
  | cond=LogicalExpression '?' then=Expression ':' else=ConditionalExpression      -> Ternary
  | cond=LogicalExpression '?' then=Expression ':' else=LambdaOrMethodReference    -> Ternary
;

%left '||';
%left '&&';
%left '|';
%left '^';
%left '&';
%left '==' '!=';

LogicalExpression -> Expression :
    RelationalExpression
  | left=LogicalExpression '||' right=LogicalExpression               -> Or
  | left=LogicalExpression '&&' right=LogicalExpression               -> And
  | left=LogicalExpression '|' right=LogicalExpression                -> BitOr
  | left=LogicalExpression '^' right=LogicalExpression                -> BitXor
  | left=LogicalExpression '&' right=LogicalExpression                -> BitAnd
  | left=LogicalExpression '==' right=LogicalExpression               -> Equality
  | left=LogicalExpression '!=' right=LogicalExpression               -> Inequality
;

# Note: relational expressions are not associative, so we've rewritten the
# grammar here to disallow parsing of 'a<b>c' as two comparisons.
RelationalExpression -> Expression :
    ArithmeticExpression
  | left=ArithmeticExpressionNoName '<' right=ArithmeticExpression    -> Relational
  | (left=QualifiedName -> ExprName) (?= !MethodReferenceLookahead) '<' right=ArithmeticExpression  -> Relational
  | left=ArithmeticExpression '>' right=ArithmeticExpression          -> Relational
  | left=ArithmeticExpression '<=' right=ArithmeticExpression         -> Relational
  | left=ArithmeticExpression '>=' right=ArithmeticExpression         -> Relational
  | RelationalExpression 'instanceof' ReferenceType                   -> InstanceOf
;

%left '<<' '>>' '>>>';
%left '+' '-';
%left '*' '/' '%';

ArithmeticExpression -> Expression :
    ArithmeticExpressionNoName
  | QualifiedName                                          -> ExprName
;

ArithmeticExpressionNoName -> Expression :
    UnaryExpressionNoName
  | left=ArithmeticExpression '<<' right=ArithmeticExpression         -> Shift
  | left=ArithmeticExpression '>>' right=ArithmeticExpression         -> Shift
  | left=ArithmeticExpression '>>>' right=ArithmeticExpression        -> Shift
  | left=ArithmeticExpression '+' right=ArithmeticExpression          -> Additive
  | left=ArithmeticExpression '-' right=ArithmeticExpression          -> Additive
  | left=ArithmeticExpression '*' right=ArithmeticExpression          -> Multiplicative
  | left=ArithmeticExpression '/' right=ArithmeticExpression          -> Multiplicative
  | left=ArithmeticExpression '%' right=ArithmeticExpression          -> Multiplicative
;

UnaryExpression -> Expression :
    UnaryExpressionNoName
  | QualifiedName                                          -> ExprName
;

UnaryExpressionNoName -> Expression :
    PreIncrementExpression
  | PreDecrementExpression
  | '+' UnaryExpression                                    -> Unary
  | '-' UnaryExpression                                    -> Unary
  | UnaryExpressionNotPlusMinusNoName
;

PreIncrementExpression -> PreInc :
    '++' UnaryExpression ;

PreDecrementExpression -> PreDec :
    '--' UnaryExpression ;

UnaryExpressionNotPlusMinus -> Expression :
    UnaryExpressionNotPlusMinusNoName
  | QualifiedName                                          -> ExprName
;

UnaryExpressionNotPlusMinusNoName -> Expression :
    PostfixExpressionNoName
  | '~' UnaryExpression                                    -> Unary
  | '!' UnaryExpression                                    -> Unary
  | CastExpression
;

PostfixExpression -> Expression :
    PostfixExpressionNoName
  | QualifiedName                                          -> ExprName
;

PostfixExpressionNoName -> Expression :
    Primary
  | PostIncrementExpression
  | PostDecrementExpression
;

PostIncrementExpression -> PostInc :
    PostfixExpression '++' ;

PostDecrementExpression -> PostDec :
    PostfixExpression '--' ;

CastStartLookahead :
    PrimitiveType ')' set(first UnaryExpression)
  | ReferenceType AdditionalBound+? ')' set(first AfterReferenceTypeCast)
;

AfterReferenceTypeCast :
    UnaryExpressionNotPlusMinus
  | LambdaExpression
;

CastExpression :
    '(' (?= CastStartLookahead & !LambdaStartLookahead) PrimitiveType ')' UnaryExpression -> CastExpression
  | '(' (?= CastStartLookahead & !LambdaStartLookahead) ReferenceType AdditionalBound+? ')' UnaryExpressionNotPlusMinusNoName -> CastExpression
  | ('(' (?= CastStartLookahead & !LambdaStartLookahead) ReferenceType AdditionalBound+? ')' (QualifiedName -> ExprName) -> CastExpression) (?= !MethodReferenceLookahead)
;

%%

${template go_lexer.stateVars}
  depth        int
  token        Token // last token
  ordinaryUnit bool
  moduleUnit   bool
${end}

${template go_lexer.initStateVars-}
  l.depth = 0
  l.token = UNAVAILABLE
  l.ordinaryUnit = false
  l.moduleUnit = false
${end}

${template go_lexer.onAfterNext}
  l.token = token
${end}

${template go_parser.stateVars-}
${call base}
	lastTypeArgOffset int
	lastClassRefOffset int
	lastClassRefOffset2 int
${end}

${template go_lexer.lexer-}
${call base-}

func (l *Lexer) moduleAsID() bool {
  switch l.token {
  case ATSIGN, PACKAGE, IMPORT, STATIC, DOT:
    return true
  }
  return l.moduleUnit || l.ordinaryUnit || l.depth > 0
}

func (l *Lexer) directiveAsID() bool {
  return !l.moduleUnit || l.token != LBRACE && l.token != SEMICOLON
}
${end}
