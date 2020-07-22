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
		identWidths:    map[*syntax.Ident]int{},
		pendingComment: map[syntax.Node]commentRelocation{},
		commentReloc:   map[syntax.Node]commentRelocation{},
	}
	if err := annotateAST(ast, &ann); err != nil {
		return false, nil, fmt.Errorf("failed annotation: %v", err)
	}
	if err := relocateComments(ast, &ann); err != nil {
		return false, nil, err
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
	lastFn      syntax.Expr
}

func (o fmtOpts) AddIndent(i int) fmtOpts {
	dupe := o
	dupe.indentLevel += i
	return dupe
}

func (o fmtOpts) shouldCondense() bool {
	if expandedFunction(o.lastFn) {
		return false
	}
	return o.binNested && o.fnNested
}

func (o fmtOpts) LeadIn(b *bytes.Buffer) {
	switch o.Context {
	case "list", "dict":
		b.WriteString("\n" + strings.Repeat(" ", o.indentLevel))
	case "arg":
		if !o.shouldCondense() {
			b.WriteString("\n" + strings.Repeat(" ", o.indentLevel))
		}
	}
}

func expandedFunction(fn syntax.Expr) bool {
	switch v := fn.(type) {
	case *syntax.Ident:
		switch v.Name {
		case "deb", "generator", "checker":
			return true
		}
	case *syntax.DotExpr:
		if ident1, ok := v.X.(*syntax.Ident); ok {
			switch ident1.Name {
			case "step":
				return false
			}
		}
	}
	return false
}

type commentRelocation struct {
	From   syntax.Node
	Before []syntax.Comment
}

type annotations struct {
	identWidths    map[*syntax.Ident]int
	pendingComment map[syntax.Node]commentRelocation
	commentReloc   map[syntax.Node]commentRelocation
}

func relocateComments(ast syntax.Node, ann *annotations) error {
	var lastCall *syntax.CallExpr
	syntax.Walk(ast, func(n syntax.Node) bool {
		if reloc, hasReloc := ann.pendingComment[n]; hasReloc {
			if lastCall != nil {
				ann.commentReloc[lastCall] = reloc
				delete(ann.pendingComment, n)
				n.Comments().Before = nil
			}
		}
		if c, ok := n.(*syntax.CallExpr); ok {
			lastCall = c
		}
		return true
	})
	return nil
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
	case *syntax.DotExpr:
		if err := annotateAST(n.X, ann); err != nil {
			return err
		}
		if err := annotateAST(n.Name, ann); err != nil {
			return err
		}

	case *syntax.DictEntry:
		if err := annotateAST(n.Key, ann); err != nil {
			return err
		}
		if err := annotateAST(n.Value, ann); err != nil {
			return err
		}

	case *syntax.Ident, *syntax.Literal:

	case *syntax.ListExpr:
		for _, l := range n.List {
			if err := annotateAST(l, ann); err != nil {
				return err
			}
		}
	case *syntax.DictExpr:
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
		maybeAnnotateCall(n, ann)
		for _, arg := range n.Args {
			if err := annotateAST(arg, ann); err != nil {
				return err
			}
		}

	case *syntax.ExprStmt:
		if err := annotateAST(n.X, ann); err != nil {
			return err
		}
		// Comments within the parenthesis of a call, but after all arguments are
		// incorrectly associated with the next ExprStmt. If we detect this, mark
		// the node as such, to be moved by relocateComments().
		if c := n.Comments(); !preceedingCommentInCall(n, c) {
			ann.pendingComment[n] = commentRelocation{Before: c.Before, From: n}
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

func preceedingCommentInCall(e syntax.Node, c *syntax.Comments) bool {
	if c != nil && len(c.Before) > 0 {
		if es, _ := e.Span(); es.Line > c.Before[0].Start.Line {
			return false
		}
	}
	return true
}

func fmtAST(ast syntax.Node, b *bytes.Buffer, opts fmtOpts) error {
	opts.LeadIn(b)
	// fmt.Printf("++%T: %+v\n", ast, ast)
	// defer func() {
	// 	fmt.Printf("--leaving: %T\n", ast)
	// }()

	if c := ast.Comments(); c != nil {
		for _, c := range c.Before {
			b.WriteString(fmtComment(c.Text))
			b.WriteString("\n" + strings.Repeat(" ", opts.indentLevel))
		}
	}

	if opts.Context == "arg" || opts.Context == "list" || opts.Context == "dict" {
		opts.Context = ""
	}

	switch n := ast.(type) {
	case *syntax.DotExpr:
		if err := fmtAST(n.X, b, opts); err != nil {
			return err
		}
		b.WriteRune('.')
		if err := fmtAST(n.Name, b, opts); err != nil {
			return err
		}

	case *syntax.Literal:
		b.WriteString(n.Raw)

	case *syntax.Ident:
		b.WriteString(n.Name)
		if w := opts.annotations.identWidths[n]; w > 0 && !opts.shouldCondense() {
			b.WriteString(strings.Repeat(" ", w-len(n.Name)))
		}

	case *syntax.DictEntry:
		if err := fmtAST(n.Key, b, opts); err != nil {
			return err
		}
		b.WriteString(": ")
		if err := fmtAST(n.Value, b, opts); err != nil {
			return err
		}

	case *syntax.DictExpr:
		b.WriteString("{")
		for i, l := range n.List {
			argOpts := opts.AddIndent(2)
			argOpts.Context = "dict"
			if err := fmtAST(l, b, argOpts); err != nil {
				return err
			}
			b.WriteString(",")
			if i == len(n.List)-1 {
				b.WriteString("\n" + strings.Repeat(" ", opts.indentLevel))
			}
		}
		b.WriteString("}")

	case *syntax.ListExpr:
		b.WriteString("[")
		for i, l := range n.List {
			argOpts := opts.AddIndent(2)
			argOpts.Context = "list"
			if err := fmtAST(l, b, argOpts); err != nil {
				return err
			}
			b.WriteString(",")
			if i == len(n.List)-1 {
				b.WriteString("\n" + strings.Repeat(" ", opts.indentLevel))
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
		condense := opts.shouldCondense() && !expandedFunction(n.Fn)

		for i, arg := range n.Args {
			argOpt := opts.AddIndent(2)
			argOpt.Context = "arg"
			argOpt.lastFn = n.Fn
			argOpt.fnNested = true
			if err := fmtAST(arg, b, argOpt); err != nil {
				return err
			}
			if !condense || i < len(n.Args)-1 {
				b.WriteString(",")
				if condense {
					b.WriteRune(' ')
				}
			}
			if bin, ok := arg.(syntax.Node); ok {
				if c := bin.Comments(); c != nil {
					for _, comment := range c.Suffix {
						b.WriteString(" " + fmtComment(comment.Text))
					}
				}
			}
		}

		if reloc, hasCommentReloc := opts.annotations.commentReloc[n]; hasCommentReloc {
			for _, comment := range reloc.Before {
				b.WriteString("\n" + strings.Repeat(" ", opts.indentLevel+1))
				b.WriteString(" " + fmtComment(comment.Text))
			}
		}

		if !condense && len(n.Args) > 0 {
			b.WriteString("\n" + strings.Repeat(" ", opts.indentLevel))
			b.WriteString(")")
		} else {
			b.WriteString(")")
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
			b.WriteString("\n")
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
