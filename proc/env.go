package proc

import (
	"encoding/gob"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/docker/docker/pkg/reexec"
)

const cmdTimeout = 5 * time.Second

// Env represents an environment in which proceedures or builds can be run.
type Env struct {
	dir       string
	artifacts string

	p            *exec.Cmd
	cmdW, cmdR   *os.File
	respW, respR *os.File
	enc          *gob.Encoder
	dec          *gob.Decoder
}

// RunBlocking runs the specified command.
func (e *Env) RunBlocking(args ...string) error {
	return e.sendCommand(procCommand{Code: cmdRunBlocking, Args: args})
}

// dependenciesInstalled returns true if dependencies have been installed.
func dependenciesInstalled() (bool, error) {
	if _, err := exec.LookPath("fuse-overlayfs"); err != nil {
		return false, nil
	}
	return true, nil
}

func NewEnv(readOnly bool) (*Env, error) {
	good, err := dependenciesInstalled()
	if err != nil {
		return nil, err
	}
	if !good {
		return nil, errors.New("host system does not have fuse-overlayfs installed")
	}

	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, err
	}

	out := Env{dir: tmp, artifacts: filepath.Join(tmp, "artifacts")}
	if err := os.Mkdir(out.artifacts, 0755); err != nil {
		os.RemoveAll(tmp)
		return nil, err
	}

	if out.cmdR, out.cmdW, err = os.Pipe(); err != nil {
		os.RemoveAll(tmp)
		return nil, err
	}
	if out.respR, out.respW, err = os.Pipe(); err != nil {
		out.cmdW.Close()
		out.cmdR.Close()
		os.RemoveAll(tmp)
		return nil, err
	}

	out.p = reexec.Command("reexecEntry", "env", strconv.FormatBool(readOnly))
	out.p.Stderr = os.Stderr
	out.p.Stdout = os.Stdout
	out.p.Stdin = os.Stdin
	out.p.Dir = out.dir
	out.p.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWUTS | syscall.CLONE_NEWUSER,
		UidMappings: []syscall.SysProcIDMap{
			{
				HostID: os.Getuid(),
				Size:   1,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				HostID: os.Getgid(),
				Size:   1,
			},
		},
		Setpgid: true,
		Pgid:    0,
	}
	out.p.ExtraFiles = []*os.File{out.cmdR, out.respW}

	if err := out.p.Start(); err != nil {
		out.cmdW.Close()
		out.cmdR.Close()
		out.respW.Close()
		out.respR.Close()
		os.RemoveAll(tmp)
		return nil, err
	}

	out.enc = gob.NewEncoder(out.cmdW)
	out.dec = gob.NewDecoder(out.respR)
	return &out, nil
}

func (e *Env) Close() error {
	err := e.sendCommand(procCommand{Code: cmdShutdown})
	if err2 := e.p.Process.Kill(); err == nil {
		err = err2
	}
	if err2 := e.cmdW.Close(); err == nil {
		err = err2
	}
	if err2 := e.cmdR.Close(); err == nil {
		err = err2
	}
	if err2 := e.respW.Close(); err == nil {
		err = err2
	}
	if err2 := e.respR.Close(); err == nil {
		err = err2
	}
	if err2 := e.p.Wait(); err == nil {
		if ee, isEE := err2.(*exec.ExitError); !isEE || ee.Error() != "signal: killed" {
			err = err2
		}
	}
	if err2 := os.RemoveAll(e.dir); err == nil {
		err = err2
	}
	return err
}
