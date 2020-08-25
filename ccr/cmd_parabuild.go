package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"sync"

	"github.com/twitchylinux/ccr"
	"github.com/twitchylinux/ccr/gen"
	"github.com/twitchylinux/ccr/log"
	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/common"
)

var (
	planOnly            = flag.Bool("plan", false, "Print the build plan then exit. Only valid for the para-build command.")
	numParabuildWorkers = flag.Int("workers", 3, "Number of workers. Only valid fro the para-build command.")
)

func doParabuildCmd(target string) error {
	console := &log.Console{}
	uv := ccr.NewUniverse(console, resCache)

	dr := ccr.NewDirResolver(*dir)
	findOpts := ccr.FindOptions{
		FallbackResolvers: []ccr.CCRResolver{dr.Resolve},
		PrefixResolvers: map[string]ccr.CCRResolver{
			"common": common.Resolve,
		},
	}

	if err := uv.Build([]vts.TargetRef{{Path: target}}, &findOpts, *baseDir); err != nil {
		return err
	}

	out, err := uv.TargetsDependencyOrder(ccr.GenerateConfig{}, vts.TargetRef{Path: target}, *baseDir, vts.TargetBuild)
	if err != nil {
		return err
	}

	if err := printBuildPlan(uv, out); err != nil {
		return err
	}
	if *planOnly {
		return nil
	}

	for i := range out {
		fmt.Printf("\033[1;31mCommencing phase %d\033[0m\n", i+1)

		var (
			wg   sync.WaitGroup
			work = make(chan *vts.Build)
			errC = make(chan error, 200)
		)
		wg.Add(*numParabuildWorkers)
		for n := 0; n < *numParabuildWorkers; n++ {
			go buildWorker(&wg, uv, target, work, errC, console)
		}

		for _, t := range out[i] {
			select {
			case err := <-errC:
				close(work)
				return err
			case work <- t.(*vts.Build):
			}
		}
		close(work)
		wg.Wait()
	}

	return nil
}

func buildWorker(wg *sync.WaitGroup, uv *ccr.Universe, target string, work chan *vts.Build, errC chan error, console *log.Console) {
	defer wg.Done()
	env := uv.MakeEnv(*baseDir)

	for target := range work {
		gc := gen.GenerationContext{
			Cache:     resCache,
			RunnerEnv: env,
			Console:   console,
		}
		if err := gen.Generate(gc, target); err != nil {
			errC <- fmt.Errorf("generate failed: %v", err)
			return
		}
	}
}

func printBuildPlan(uv *ccr.Universe, out [][]vts.Target) error {
	for i, phase := range out {
		fmt.Printf("\033[1;34mPhase %03d\033[0m (%d builds):\n", i+1, len(phase))
		for _, b := range phase {
			build := b.(*vts.Build)
			h, err := uv.TargetRollupHash(build.GlobalPath())
			if err != nil {
				return err
			}

			mark := "\033[1;31m✖\033[0m"
			if cached, _ := resCache.IsHashCached(h); cached {
				mark = "\033[1;32m✓\033[0m"
			}

			fmt.Printf("  %s [%s] \033[1;33m%s\033[0m\n", mark, base64.RawURLEncoding.EncodeToString(h), build.GlobalPath())
		}
		fmt.Println()
	}
	return nil
}
