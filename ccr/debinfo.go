package main

import (
	"bytes"
	"fmt"
	"io"
	"strconv"

	d2 "github.com/twitchyliquid64/debdep/deb"
)

func mkDebSource(baseURL string, p *d2.Paragraph) (*bytes.Buffer, error) {
	var out bytes.Buffer

	fmt.Fprintf(&out, "deb(\n")
	fmt.Fprintf(&out, "  name    = %s,\n", strconv.Quote("debsrc_"+p.Values["Package"]))
	fmt.Fprintf(&out, "  url     = %s,\n", strconv.Quote(baseURL+"/"+p.Values["Filename"]))
	fmt.Fprintf(&out, "  sha256  = %s,\n", strconv.Quote(p.Values["SHA256"]))
	fmt.Fprintf(&out, "  details = [\n")
	fmt.Fprintf(&out, "    attr(parent = 'common://attrs:deb_info', value = {\n")
	fmt.Fprintf(&out, "      'name': %s,\n", strconv.Quote(p.Values["Package"]))
	fmt.Fprintf(&out, "      'version': %s,\n", strconv.Quote(p.Values["Version"]))
	if _, hasMaintainer := p.Values["Maintainer"]; hasMaintainer {
		fmt.Fprintf(&out, "      'maintainer': %s,\n", strconv.Quote(p.Values["Maintainer"]))
	}
	if _, hasDesc := p.Values["Description"]; hasDesc {
		fmt.Fprintf(&out, "      'description': %s,\n", strconv.Quote(p.Values["Description"]))
	}
	if _, hasHomepage := p.Values["Homepage"]; hasHomepage {
		fmt.Fprintf(&out, "      'homepage': %s,\n", strconv.Quote(p.Values["Homepage"]))
	}
	if _, hasSection := p.Values["Section"]; hasSection {
		fmt.Fprintf(&out, "      'section': %s,\n", strconv.Quote(p.Values["Section"]))
	}
	if _, hasPriority := p.Values["Priority"]; hasPriority {
		fmt.Fprintf(&out, "      'priority': %s,\n", strconv.Quote(p.Values["Priority"]))
	}
	if err := debDependsDescription(&out, p, "depends-on"); err != nil {
		return nil, err
	}
	if err := debPreDependsDescription(&out, p, "pre-depends"); err != nil {
		return nil, err
	}
	fmt.Fprintf(&out, "    }),\n")
	fmt.Fprintf(&out, "  ],\n")
	fmt.Fprintf(&out, ")")

	return &out, nil
}

func debPreDependsDescription(final *bytes.Buffer, p *d2.Paragraph, key string) error {
	dep, err := p.BinaryPreDepends()
	if err != nil {
		return err
	}
	var out bytes.Buffer

	fmt.Fprintf(&out, "      'pre-depends-on': [\n")
	if err := debRelationDescription(&out, dep); err != nil {
		return err
	}
	fmt.Fprintf(&out, "      ],\n")

	if _, err := io.Copy(final, &out); err != nil {
		return err
	}
	return nil
}

func debDependsDescription(final *bytes.Buffer, p *d2.Paragraph, key string) error {
	dep, err := p.BinaryDepends()
	if err != nil {
		return err
	}
	var out bytes.Buffer

	fmt.Fprintf(&out, "      'depends-on': [\n")
	if err := debRelationDescription(&out, dep); err != nil {
		return err
	}
	fmt.Fprintf(&out, "      ],\n")

	if _, err := io.Copy(final, &out); err != nil {
		return err
	}
	return nil
}

func debRelationDescription(final *bytes.Buffer, dep d2.Requirement) error {
	var out bytes.Buffer
	switch dep.Kind {
	case d2.PackageRelationRequirement:
		fmt.Fprintf(&out, "        {\n")
		fmt.Fprintf(&out, "          'name': %s,\n", strconv.Quote(dep.Package))
		if dep.VersionConstraint != nil {
			fmt.Fprintf(&out, "          'version': %s,\n", strconv.Quote(dep.VersionConstraint.Version))
			fmt.Fprintf(&out, "          'version-constraint': %s,\n", strconv.Quote(string(dep.VersionConstraint.ConstraintRelation)))
		}
		fmt.Fprintf(&out, "        },\n")
	case d2.AndCompositeRequirement:
		for i := range dep.Children {
			if err := debRelationDescription(&out, dep.Children[i]); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("cannot handle relation type: %v", dep)
	}

	if _, err := io.Copy(final, &out); err != nil {
		return err
	}
	return nil
}
