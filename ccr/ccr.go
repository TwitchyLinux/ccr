// Binary ccr is the Core Contracts resolver utility.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

var (
	inline  = flag.Bool("i", false, "When formatting, update files inline.")
	dir     = flag.String("contracts-dir", "", "Use the provided directory when reading contracts instead of the working directory.")
	baseDir = flag.String("base-dir", "", "Use the provided directory as the base directory instead of the working directory.")
)

func main() {
	flag.Parse()

	if *baseDir == "" {
		wd, _ := os.Getwd()
		*baseDir = wd
	}

	if *inline && flag.Arg(0) != "fmt" {
		fmt.Fprintf(os.Stderr, "Error: %s\n", "--inline may only be specified with the fmt sub-command.")
		os.Exit(1)
	}

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	switch flag.Arg(0) {
	case "fmt":
		return doFmtCmd(flag.Args()[1:])
	case "lint":
		return doLintCmd(flag.Args()[1:])
	case "coverage":
		return doCoverageCmd()
	case "check":
		return doCheckCmd()
	case "":
		fmt.Fprintf(os.Stderr, "Error: Expected command \"fmt\", \"lint\", \"check\", or \"generate\".\n")
		os.Exit(1)
	default:
		fmt.Fprintf(os.Stderr, "Error: Unknown command %q.\n", flag.Arg(0))
		os.Exit(1)
	}
	return nil
}

type file struct {
	path string
	mode os.FileMode
}

func filesInPaths(paths []string) ([]file, error) {
	var out []file
	if len(paths) == 0 {
		paths = []string{"."}
	}

	var recurseTargets []string
	for _, f := range paths {
		s, err := os.Stat(f)
		if err != nil {
			return nil, err
		}
		if !s.IsDir() {
			out = append(out, file{f, s.Mode()})
			continue
		}

		contents, err := ioutil.ReadDir(f)
		if err != nil {
			return nil, err
		}
		for _, c := range contents {
			recurseTargets = append(recurseTargets, filepath.Join(f, c.Name()))
		}
	}

	if len(recurseTargets) > 0 {
		extra, err := filesInPaths(recurseTargets)
		if err != nil {
			return nil, err
		}
		out = append(out, extra...)
	}
	return out, nil
}
