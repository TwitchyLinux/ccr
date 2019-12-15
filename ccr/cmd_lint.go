package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/twitchylinux/ccr/ccr/pretty"
)

func doLintCmd(paths []string) error {
	files, err := filesInPaths(paths)
	if err != nil {
		return err
	}

	anyChanged := false
	for _, f := range files {
		if !strings.HasSuffix(f.path, ".ccr") {
			continue
		}
		changed, _, err := pretty.FormatCCR(f.path)
		if err != nil {
			return err
		}
		anyChanged = anyChanged || changed
		if changed {
			fmt.Println(f.path)
		}
	}

	if anyChanged {
		os.Exit(1)
	}
	return nil
}
