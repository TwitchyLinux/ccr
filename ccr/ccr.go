// Binary ccr is the Core Contracts resolver utility.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/twitchylinux/ccr/ccr/pretty"
)

var (
	inline = flag.Bool("i", false, "When formatting, update files inline.")
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	switch flag.Arg(0) {
	case "fmt":
		return doFmtCmd(flag.Args()[1:])
	case "":
		fmt.Fprintf(os.Stderr, "Error: Expected command \"fmt\" or xxxx.\n")
		os.Exit(1)
	default:
		fmt.Fprintf(os.Stderr, "Error: Unknown command %q.\n", flag.Arg(0))
		os.Exit(1)
	}
	return nil
}

func doFmtCmd(paths []string) error {
	if len(paths) == 0 {
		paths = []string{"."}
	}

	for _, f := range paths {
		s, err := os.Stat(f)
		if err != nil {
			return err
		}
		if s.IsDir() {
			return fmt.Errorf("cannot format directory %q: not implemented", f)
		}

		changed, out, err := pretty.FormatCCR(f)
		if err != nil {
			return err
		}
		if changed {
			if *inline {
				if err := ioutil.WriteFile(f, out.Bytes(), s.Mode()&os.ModePerm); err != nil {
					return err
				}
			} else {
				if _, err := os.Stdout.Write(out.Bytes()); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
