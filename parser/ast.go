// Copyright 2016 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package parser

import (
	"fmt"
	"strings"
	"text/scanner"
)

// Definition is an Assignment or a Module at the top level of a Blueprints file
type Definition interface {
	String() string
	definitionTag()
}

// An Assignment is a variable assignment at the top level of a Blueprints file, scoped to the
// file and and subdirs.
type Assignment struct {
	Name       string
	NamePos    scanner.Position
	Value      Expression
	OrigValue  Expression
	EqualsPos  scanner.Position
	Assigner   string
	Referenced bool
}

func (a *Assignment) String() string {
	return fmt.Sprintf("%s@%s %s %s (%s) %t", a.Name, a.EqualsPos, a.Assigner, a.Value, a.OrigValue, a.Referenced)
}

func (a *Assignment) definitionTag() {}

// A Module is a module definition at the top level of a Blueprints file
type Module struct {
	Type    string
	TypePos scanner.Position
	Map
}

func (m *Module) Copy() *Module {
	ret := *m
	ret.Properties = make([]*Property, len(m.Properties))
	for i := range m.Properties {
		ret.Properties[i] = m.Properties[i].Copy()
	}
	return &ret
}

func (m *Module) String() string {
	propertyStrings := make([]string, len(m.Properties))
	for i, property := range m.Properties {
		propertyStrings[i] = property.String()
	}
	return fmt.Sprintf("%s@%s-%s{%s}", m.Type,
		m.LBracePos, m.RBracePos,
		strings.Join(propertyStrings, ", "))
}

func (m *Module) definitionTag() {}

func (m *Module) Pos() scanner.Position { return m.TypePos }
func (m *Module) End() scanner.Position { return m.Map.End() }

// A Property is a name: value pair within a Map, which may be a top level Module.
type Property struct {
	Name     string
	NamePos  scanner.Position
	ColonPos scanner.Position
	Value    Expression
}

func (p *Property) Copy() *Property {
	ret := *p
	ret.Value = p.Value.Copy()
	return &ret
}

func (p *Property) String() string {
	return fmt.Sprintf("%s@%s: %s", p.Name, p.ColonPos, p.Value)
}

// An Expression is a Value in a Property or Assignment.  It can be a literal (String or Bool), a
// Map, a List, an Operator that combines two expressions of the same type, or a Variable that
// references and Assignment.
type Expression interface {
	// Copy returns a copy of the Expression that will not affect the original if mutated
	Copy() Expression
	String() string
	// Type returns the underlying Type enum of the Expression if it were to be evalutated
	Type() Type
	// Pos returns the position of the first token in the Expression
	Pos() scanner.Position
	// End returns the position of the beginning of the last token in the Expression
	End() scanner.Position
	// Eval returns an expression that is fully evaluated to a simple type (List, Map, String, or
	// Bool).  It will return the same object for every call to Eval().
	Eval() Expression
}

type Type int

const (
	BoolType Type = iota + 1
	StringType
	ListType
	MapType
)

func (t Type) String() string {
	switch t {
	case BoolType:
		return "bool"
	case StringType:
		return "string"
	case ListType:
		return "list"
	case MapType:
		return "map"
	default:
		panic(fmt.Errorf("Unknown type %d", t))
	}
}

type Operator struct {
	Args        [2]Expression
	Operator    rune
	OperatorPos scanner.Position
	Value       Expression
}

func (x *Operator) Copy() Expression {
	ret := *x
	ret.Args[0] = x.Args[0].Copy()
	ret.Args[1] = x.Args[1].Copy()
	return &ret
}

func (x *Operator) Eval() Expression {
	return x.Value.Eval()
}

func (x *Operator) Type() Type {
	return x.Args[0].Type()
}

func (x *Operator) Pos() scanner.Position { return x.Args[0].Pos() }
func (x *Operator) End() scanner.Position { return x.Args[1].End() }

func (x *Operator) String() string {
	return fmt.Sprintf("(%s %c %s = %s)@%s", x.Args[0].String(), x.Operator, x.Args[1].String(),
		x.Value, x.OperatorPos)
}

type Variable struct {
	Name    string
	Value   Expression
	NamePos scanner.Position
}

func (x *Variable) Pos() scanner.Position { return x.NamePos }
func (x *Variable) End() scanner.Position { return x.NamePos }

func (x *Variable) Copy() Expression {
	ret := *x
	return &ret
}

func (x *Variable) Eval() Expression {
	return x.Value.Eval()
}

func (x *Variable) String() string {
	return x.Name + " = " + x.Value.String()
}

func (x *Variable) Type() Type { return x.Value.Type() }

type Map struct {
	LBracePos  scanner.Position
	RBracePos  scanner.Position
	Properties []*Property
}

func (x *Map) Pos() scanner.Position { return x.LBracePos }
func (x *Map) End() scanner.Position { return x.RBracePos }

func (x *Map) Copy() Expression {
	ret := *x
	ret.Properties = make([]*Property, len(x.Properties))
	for i := range x.Properties {
		ret.Properties[i] = x.Properties[i].Copy()
	}
	return &ret
}

func (x *Map) Eval() Expression {
	return x
}

func (x *Map) String() string {
	propertyStrings := make([]string, len(x.Properties))
	for i, property := range x.Properties {
		propertyStrings[i] = property.String()
	}
	return fmt.Sprintf("@%s-%s{%s}", x.LBracePos, x.RBracePos,
		strings.Join(propertyStrings, ", "))
}

func (x *Map) Type() Type { return MapType }

type List struct {
	LBracePos scanner.Position
	RBracePos scanner.Position
	Values    []Expression
}

func (x *List) Pos() scanner.Position { return x.LBracePos }
func (x *List) End() scanner.Position { return x.RBracePos }

func (x *List) Copy() Expression {
	ret := *x
	ret.Values = make([]Expression, len(x.Values))
	for i := range ret.Values {
		ret.Values[i] = x.Values[i].Copy()
	}
	return &ret
}

func (x *List) Eval() Expression {
	return x
}

func (x *List) String() string {
	valueStrings := make([]string, len(x.Values))
	for i, value := range x.Values {
		valueStrings[i] = value.String()
	}
	return fmt.Sprintf("@%s-%s[%s]", x.LBracePos, x.RBracePos,
		strings.Join(valueStrings, ", "))
}

func (x *List) Type() Type { return ListType }

type String struct {
	LiteralPos scanner.Position
	Value      string
}

func (x *String) Pos() scanner.Position { return x.LiteralPos }
func (x *String) End() scanner.Position { return x.LiteralPos }

func (x *String) Copy() Expression {
	ret := *x
	return &ret
}

func (x *String) Eval() Expression {
	return x
}

func (x *String) String() string {
	return fmt.Sprintf("%q@%s", x.Value, x.LiteralPos)
}

func (x *String) Type() Type {
	return StringType
}

type Bool struct {
	LiteralPos scanner.Position
	Value      bool
}

func (x *Bool) Pos() scanner.Position { return x.LiteralPos }
func (x *Bool) End() scanner.Position { return x.LiteralPos }

func (x *Bool) Copy() Expression {
	ret := *x
	return &ret
}

func (x *Bool) Eval() Expression {
	return x
}

func (x *Bool) String() string {
	return fmt.Sprintf("%t@%s", x.Value, x.LiteralPos)
}

func (x *Bool) Type() Type {
	return BoolType
}

type Comment struct {
	Comment []string
	Slash   scanner.Position
}

func (c Comment) Pos() scanner.Position {
	return c.Slash
}

func (c Comment) End() scanner.Position {
	pos := c.Slash
	for _, comment := range c.Comment {
		pos.Offset += len(comment)
	}
	pos.Line += len(c.Comment) - 1
	return pos
}

func (c Comment) String() string {
	l := 0
	for _, comment := range c.Comment {
		l += len(comment) + 1
	}
	buf := make([]byte, 0, l)
	for _, comment := range c.Comment {
		buf = append(buf, comment...)
		buf = append(buf, '\n')
	}

	return string(buf) + "@" + c.Slash.String()
}

// Return the text of the comment with // or /* and */ stripped
func (c Comment) Text() string {
	l := 0
	for _, comment := range c.Comment {
		l += len(comment) + 1
	}
	buf := make([]byte, 0, l)

	blockComment := false
	if strings.HasPrefix(c.Comment[0], "/*") {
		blockComment = true
	}

	for i, comment := range c.Comment {
		if blockComment {
			if i == 0 {
				comment = strings.TrimPrefix(comment, "/*")
			}
			if i == len(c.Comment)-1 {
				comment = strings.TrimSuffix(comment, "*/")
			}
		} else {
			comment = strings.TrimPrefix(comment, "//")
		}
		buf = append(buf, comment...)
		buf = append(buf, '\n')
	}

	return string(buf)
}
