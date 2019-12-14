// Package pretty formats .ccr files.
package pretty

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"

	"go.starlark.net/syntax"
)

// FormatCCR reads the provided file, generating a formatted representation.
// False is returned if no modification was made.
func FormatCCR(fPath string) (bool, *bytes.Buffer, error) {
	d, err := ioutil.ReadFile(fPath)
	if err != nil {
		return false, nil, err
	}

	ast, err := syntax.Parse(fPath, d, syntax.RetainComments)
	if err != nil {
		return false, nil, err
	}
	var b bytes.Buffer
	b.Grow(1024)

	ann := annotations{
		identWidths: map[*syntax.Ident]int{},
	}
	if err := annotateAST(ast, &ann); err != nil {
		return false, nil, fmt.Errorf("failed annotation: %v", err)
	}
	if err := fmtAST(ast, &b, fmtOpts{annotations: &ann}); err != nil {
		return false, nil, err
	}

	return !bytes.Equal(b.Bytes(), d), &b, nil
}

type fmtOpts struct {
	indentLevel int
	annotations *annotations
	Context     string
	fnNested    bool
	binNested   bool
}

func (o fmtOpts) AddIndent(i int) fmtOpts {
	dupe := o
	dupe.indentLevel += i
	return dupe
}

func (o fmtOpts) shouldCondense() bool {
	return o.binNested && o.fnNested
}

func (o fmtOpts) LeadIn(b *bytes.Buffer) {
	switch o.Context {
	case "arg":
		if !o.shouldCondense() {
			b.WriteString("\n" + strings.Repeat(" ", o.indentLevel))
		}
	}
}

type annotations struct {
	identWidths map[*syntax.Ident]int
}

func maybeAnnotateCall(c *syntax.CallExpr, ann *annotations) {
	var maxWidth int
	for _, arg := range c.Args {
		b, isBin := arg.(*syntax.BinaryExpr)
		if !isBin {
			return
		}
		k, isIdent := b.X.(*syntax.Ident)
		if !isIdent {
			return
		}

		if len(k.Name) > maxWidth {
			maxWidth = len(k.Name)
		}
	}

	for _, arg := range c.Args {
		ann.identWidths[arg.(*syntax.BinaryExpr).X.(*syntax.Ident)] = maxWidth
	}
}

func annotateAST(ast syntax.Node, ann *annotations) error {
	switch n := ast.(type) {
	case *syntax.Ident, *syntax.Literal:

	case *syntax.ListExpr:
		for _, l := range n.List {
			if err := annotateAST(l, ann); err != nil {
				return err
			}
		}

	case *syntax.BinaryExpr:
		if err := annotateAST(n.X, ann); err != nil {
			return err
		}
		if err := annotateAST(n.Y, ann); err != nil {
			return err
		}

	case *syntax.CallExpr:
		for _, arg := range n.Args {
			if err := annotateAST(arg, ann); err != nil {
				return err
			}
		}

	case *syntax.ExprStmt:
		if c, isCall := n.X.(*syntax.CallExpr); isCall {
			maybeAnnotateCall(c, ann)
		}
		if err := annotateAST(n.X, ann); err != nil {
			return err
		}
	case *syntax.File:
		for _, stmt := range n.Stmts {
			if err := annotateAST(stmt, ann); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("cannot handle AST node %T (%+v)", ast, n)
	}

	return nil
}

func fmtAST(ast syntax.Node, b *bytes.Buffer, opts fmtOpts) error {
	opts.LeadIn(b)

	if c := ast.Comments(); c != nil {
		for _, c := range c.Before {
			b.WriteString(fmtComment(c.Text))
			b.WriteString("\n" + strings.Repeat(" ", opts.indentLevel))
			if opts.shouldCondense() {
				b.WriteString("  ")
			}
		}
	}

	if opts.Context == "arg" {
		opts.Context = ""
	}

	switch n := ast.(type) {
	case *syntax.Literal:
		b.WriteString(n.Raw)

	case *syntax.Ident:
		b.WriteString(n.Name)
		if w := opts.annotations.identWidths[n]; w > 0 {
			b.WriteString(strings.Repeat(" ", w-len(n.Name)))
		}

	case *syntax.ListExpr:
		b.WriteString("[")
		if len(n.List) > 0 {
			b.WriteString("\n" + strings.Repeat(" ", opts.indentLevel+2))
		}
		for i, l := range n.List {
			if err := fmtAST(l, b, opts); err != nil {
				return err
			}
			b.WriteString(",\n" + strings.Repeat(" ", opts.indentLevel))
			if i < len(n.List)-1 {
				b.WriteString(strings.Repeat(" ", 2))
			}
		}
		b.WriteString("]")

	case *syntax.BinaryExpr:
		if err := fmtAST(n.X, b, opts); err != nil {
			return err
		}
		b.WriteString(" " + n.Op.String() + " ")
		rhsOpts := opts
		rhsOpts.binNested = true
		if err := fmtAST(n.Y, b, rhsOpts); err != nil {
			return err
		}

	case *syntax.CallExpr:
		if err := fmtAST(n.Fn, b, opts); err != nil {
			return err
		}
		b.WriteString("(")
		if c := n.Fn.Comments(); c != nil {
			for _, c := range c.Suffix {
				b.WriteString("\n" + strings.Repeat(" ", opts.indentLevel+2))
				b.WriteString(fmtComment(c.Text))
			}
		}

		for i, arg := range n.Args {
			argOpt := opts.AddIndent(2)
			argOpt.Context = "arg"
			argOpt.fnNested = true
			if err := fmtAST(arg, b, argOpt); err != nil {
				return err
			}
			if !opts.shouldCondense() || i < len(n.Args)-1 {
				b.WriteString(",")
				if opts.shouldCondense() {
					b.WriteRune(' ')
				}
			}
			if bin, ok := arg.(*syntax.BinaryExpr); ok {
				if c := bin.Comments(); c != nil {
					for _, comment := range c.Suffix {
						b.WriteString(" " + fmtComment(comment.Text))
					}
				}
			}
		}
		if opts.shouldCondense() {
			b.WriteString(")")
		} else {
			b.WriteString("\n)\n")
		}

	case *syntax.ExprStmt:
		if err := fmtAST(n.X, b, opts); err != nil {
			return err
		}
	case *syntax.File:
		for i, stmt := range n.Stmts {
			if err := fmtAST(stmt, b, opts); err != nil {
				return err
			}
			if i < len(n.Stmts)-1 {
				b.WriteString("\n")
			}
		}
	default:
		return fmt.Errorf("cannot handle AST node %T (%+v)", ast, n)
	}

	if c := ast.Comments(); c != nil {
		if _, fileLevel := ast.(*syntax.File); fileLevel {
			b.WriteString("\n")
		}
		for _, c := range c.After {
			b.WriteString(fmtComment(c.Text))
			b.WriteString("\n" + strings.Repeat(" ", opts.indentLevel))
		}
	}

	return nil
}

func fmtComment(in string) string {
	if len(in) > 2 && in[0] == '#' && in[1] != ' ' {
		return "# " + in[1:]
	}
	return in
}
