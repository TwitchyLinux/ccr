package main

import (
	"io/ioutil"
	"os"

	"github.com/twitchylinux/ccr/ccr/pretty"
)

func doFmtCmd(paths []string) error {
	files, err := filesInPaths(paths)
	if err != nil {
		return err
	}

	for _, f := range files {
		changed, out, err := pretty.FormatCCR(f.path)
		if err != nil {
			return err
		}
		if changed {
			if *inline {
				if err := ioutil.WriteFile(f.path, out.Bytes(), f.mode&os.ModePerm); err != nil {
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
