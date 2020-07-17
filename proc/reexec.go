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

func commandChannels() (*gob.Encoder, *gob.Decoder, error) {
	instReader := os.NewFile(3, "control")
	if instReader == nil {
		return nil, nil, errors.New("fd 3 was not valid")
	}
	respWriter := os.NewFile(4, "resp")
	if respWriter == nil {
		return nil, nil, errors.New("fd 4 was not valid")
	}
	return gob.NewEncoder(respWriter), gob.NewDecoder(instReader), nil
}

func runBlocking(cmd procCommand, pivotDir string, readOnly bool) procResp {
	c := reexec.Command(append([]string{"reexecEntry", "run", pivotDir, strconv.FormatBool(readOnly)}, cmd.Args...)...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	c.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS,
	}
	errStr := ""
	if err := c.Run(); err != nil {
		errStr = err.Error()
	}
	return procResp{Code: cmd.Code, Error: errStr}
}

func envMainloop(cmdW *gob.Encoder, cmdR *gob.Decoder, readOnly bool) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	var fs fs
	fs, err = setupWriteableFS(wd)
	if err != nil {
		return err
	}
	defer fs.Close()

	for {
		var cmd procCommand
		if err := cmdR.Decode(&cmd); err != nil {
			return fmt.Errorf("reading command: %v", err)
		}

		switch cmd.Code {
		case cmdRunBlocking:
			cmdW.Encode(runBlocking(cmd, fs.Root(), readOnly))
		case cmdShutdown:
			cmdW.Encode(procResp{Code: cmd.Code})
			return nil
		default:
			return fmt.Errorf("unhandled command: %v", cmd.Code)
		}
	}
}

// isolatedMain is the program entrypoint when this binary is invoked in the isolated environment.
// Instructions for setting up the namespace are provided via the file system.
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

		cmdW, cmdR, err := commandChannels()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed setting up command channels: %v\n", err)
			os.Exit(reexecExitCode)
		}

		if err := envMainloop(cmdW, cmdR, readOnly); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(reexecExitCode)
		}
		return
	} else if len(os.Args) > 4 && os.Args[1] == "run" {
		readOnly, err := strconv.ParseBool(os.Args[3])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed parsing read-only argument: %v\n", err)
			os.Exit(reexecExitCode)
		}
		if err := setRootFS(os.Args[2], readOnly); err != nil {
			fmt.Fprintf(os.Stderr, "Failed setting up pivot root: %v\n", err)
			os.Exit(reexecExitCode)
		}
		prog := os.Args[4]
		if !filepath.IsAbs(prog) {
			p, err := exec.LookPath(prog)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Could not find %q: %v\n", prog, err)
				os.Exit(reexecExitCode)
			}
			prog = p
		}
		syscall.Exec(prog, os.Args[4:], os.Environ())
	}
}
