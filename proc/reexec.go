package proc

import (
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"

	"github.com/docker/docker/pkg/reexec"
)

const reexecExitCode = 217

func init() {
	reexec.Register("reexecEntry", isolatedMain)
	if reexec.Init() {
		os.Exit(0)
	}
}

func commandChannels() (*gob.Encoder, *gob.Decoder, *gob.Encoder, error) {
	instReader := os.NewFile(3, "control")
	if instReader == nil {
		return nil, nil, nil, errors.New("fd 3 was not valid")
	}
	respWriter := os.NewFile(4, "resp")
	if respWriter == nil {
		return nil, nil, nil, errors.New("fd 4 was not valid")
	}
	streamWriter := os.NewFile(5, "stream")
	if streamWriter == nil {
		return nil, nil, nil, errors.New("fd 5 was not valid")
	}
	return gob.NewEncoder(respWriter), gob.NewDecoder(instReader), gob.NewEncoder(streamWriter), nil
}

func envMainloop(cmdW *gob.Encoder, cmdR *gob.Decoder, respW *gob.Encoder, readOnly bool) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	var fs fs
	fs, err = setupEnvFS(wd)
	if err != nil {
		return err
	}
	defer fs.Close()

	em, err := makeExecManager(respW)
	if err != nil {
		return err
	}
	defer em.Close()

	for {
		var cmd procCommand
		if err := cmdR.Decode(&cmd); err != nil {
			return fmt.Errorf("reading command: %v", err)
		}

		switch cmd.Code {
		case cmdPing:
			cmdW.Encode(procResp{Code: cmdPing})
		case cmdRunBlocking:
			cmdW.Encode(runBlocking(cmd, fs.Root(), readOnly))
		case cmdRunStreaming:
			cmdW.Encode(em.RunStreaming(cmd, fs.Root(), readOnly))
		case cmdShutdown:
			// em.Close() can be called multiple times, so we close here as well as
			// in the defer to make sure things shut down before our invoker recieves
			// the command response.
			if err := em.Close(); err != nil {
				return err
			}
			cmdW.Encode(procResp{Code: cmd.Code})
			return nil
		default:
			return fmt.Errorf("unhandled command: %v", cmd.Code)
		}
	}
}

// isolatedMain is the program entrypoint when this binary is invoked in the isolated environment.
// Instructions for setting up the namespace are provided via the file system.
//
// This main provides two functions:
//  - 'env' mode: This is intended to be run in isolated namespaces mapped to
//    UID 0. In this mode, an isolated filesystem is established, and requests
//    over process file descriptors are serviced to create processes.
//  - 'run' mode: These processes are created from 'env' mode processes in
//    response to a request to run something. In run mode, The process
//    pivot_root()'s to a new view of the filesystem, before calling exec()
//    for the requested process.
func isolatedMain() {
	runtime.GOMAXPROCS(1)
	runtime.LockOSThread()

	// Prevent shared mounts from propergating in our namespace.
	if err := syscall.Mount("none,", "/", "none", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
		fmt.Fprintf(os.Stderr, "Private mount failed: %v\n", err)
		os.Exit(reexecExitCode)
	}

	if len(os.Args) > 2 && os.Args[1] == "env" {
		readOnly, err := strconv.ParseBool(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed parsing read-only argument: %v\n", err)
			os.Exit(reexecExitCode)
		}

		cmdW, cmdR, respW, err := commandChannels()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed setting up command channels: %v\n", err)
			os.Exit(reexecExitCode)
		}

		if err := envMainloop(cmdW, cmdR, respW, readOnly); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(reexecExitCode)
		}
		return
	} else if len(os.Args) > 5 && os.Args[1] == "run" {
		readOnly, err := strconv.ParseBool(os.Args[3])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed parsing read-only argument: %v\n", err)
			os.Exit(reexecExitCode)
		}
		pivotDir, prog, wd, args := os.Args[2], os.Args[5], os.Args[4], os.Args[5:]

		if err := setRootFS(pivotDir, readOnly); err != nil {
			fmt.Fprintf(os.Stderr, "Failed setting up pivot root: %v\n", err)
			os.Exit(reexecExitCode)
		}
		if !filepath.IsAbs(prog) {
			p, err := exec.LookPath(prog)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Could not find %q: %v\n", prog, err)
				os.Exit(reexecExitCode)
			}
			prog = p
		}
		os.Chdir(wd)
		syscall.Exec(prog, args, os.Environ())
	}
}
