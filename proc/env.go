package proc

import (
	"bytes"
	"crypto/rand"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/docker/docker/pkg/reexec"
)

const cmdTimeout = 5 * time.Second

// Env represents an isolated host environment.
type Env struct {
	dir string

	l                  sync.Mutex
	streamingProcesses map[string]envProc

	p            *exec.Cmd
	cmdW, cmdR   *os.File
	respW, respR *os.File
	stdW, stdR   *os.File
	stream       *gob.Decoder
	enc          *gob.Encoder
	dec          *gob.Decoder
}

type envProc struct {
	complete bool
	exitCode int
	error    string
	stdout   io.Writer
	stderr   io.Writer
}

// RunBlocking runs the specified command.
func (e *Env) RunBlocking(dir string, args ...string) ([]byte, []byte, int, error) {
	resp, err := e.sendCommand(procCommand{Code: cmdRunBlocking, Args: args, Dir: dir})
	return resp.Stdout, resp.Stderr, resp.ExitCode, err
}

// RunStreaming runs the specified command without blocking.
func (e *Env) RunStreaming(dir string, out, err io.Writer, args ...string) (string, error) {
	c := procCommand{Code: cmdRunStreaming, Args: args, Dir: dir}
	var rData [16]byte
	if _, err := rand.Read(rData[:]); err != nil {
		return "", err
	}
	c.ProcID = hex.EncodeToString(rData[:])

	if _, err := e.sendCommand(c); err != nil {
		return "", err
	}
	e.l.Lock()
	e.streamingProcesses[c.ProcID] = envProc{
		stdout: out,
		stderr: err,
	}
	e.l.Unlock()
	return c.ProcID, nil
}

// WaitStreaming returns when the streaming commands previously specified
// completes.
func (e *Env) WaitStreaming(id string) error {
	for {
		time.Sleep(45 * time.Millisecond)
		e.l.Lock()
		info, ok := e.streamingProcesses[id]
		e.l.Unlock()
		if !ok {
			return os.ErrNotExist
		}
		if info.complete {
			return nil
		}
	}
}

// StreamingExitStatus returns the exit code and error (if any) of the
// specified streaming command, or -1/ErrNotExist if it does not exist.
func (e *Env) StreamingExitStatus(id string) (int, error) {
	e.l.Lock()
	defer e.l.Unlock()
	info, ok := e.streamingProcesses[id]
	if !ok {
		return -1, os.ErrNotExist
	}
	var err error
	if info.error != "" {
		err = errors.New(info.error)
	}
	return info.exitCode, err
}

// EnsurePatched makes sure the top-level file or directory is mapped into the
// filesystem of the isolated environment.
func (e *Env) EnsurePatched(topLevelPathSegment string) error {
	_, err := e.sendCommand(procCommand{Code: cmdEnsureTLDWired, Dir: topLevelPathSegment})
	return err
}

// OverlayMountPath returns the path to the RW view of the system, within the isolated
// environment.
func (e *Env) OverlayMountPath() string {
	return filepath.Join(e.dir, "top")
}

// OverlayUpperPath returns the path to files the environment wrote to.
func (e *Env) OverlayUpperPath() string {
	return filepath.Join(e.dir, "u")
}

func (e *Env) ping() error {
	_, err := e.sendCommand(procCommand{Code: cmdPing})
	return err
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

	out := Env{dir: tmp, streamingProcesses: map[string]envProc{}}

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
	if out.stdR, out.stdW, err = os.Pipe(); err != nil {
		out.cmdW.Close()
		out.cmdR.Close()
		out.respW.Close()
		out.respR.Close()
		os.RemoveAll(tmp)
		return nil, err
	}

	out.p = reexec.Command("reexecEntry", "env", strconv.FormatBool(readOnly))
	out.p.Stderr = os.Stderr
	out.p.Stdout = os.Stdout
	out.p.Stdin = os.Stdin
	out.p.Dir = out.dir
	out.p.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWUTS | syscall.CLONE_NEWUSER,
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
	out.p.ExtraFiles = []*os.File{out.cmdR, out.respW, out.stdW}

	if err := out.p.Start(); err != nil {
		out.cmdW.Close()
		out.cmdR.Close()
		out.respW.Close()
		out.respR.Close()
		out.stdW.Close()
		out.stdR.Close()
		os.RemoveAll(tmp)
		return nil, err
	}

	out.enc = gob.NewEncoder(out.cmdW)
	out.dec = gob.NewDecoder(out.respR)
	out.stream = gob.NewDecoder(out.stdR)
	go out.streamToConsole()
	return &out, out.ping()
}

func (e *Env) streamToConsole() {
	for {
		var resp outputData
		if err := e.stream.Decode(&resp); err != nil {
			return
		}
		e.l.Lock()
		procInfo := e.streamingProcesses[resp.ProcID]
		e.l.Unlock()

		if resp.Complete {
			e.l.Lock()
			procInfo.complete = true
			procInfo.error = resp.Error
			procInfo.exitCode = resp.ExitCode
			e.streamingProcesses[resp.ProcID] = procInfo
			e.l.Unlock()
		} else {
			if resp.IsStderr {
				io.Copy(procInfo.stderr, bytes.NewReader(resp.Data))
			} else {
				io.Copy(procInfo.stdout, bytes.NewReader(resp.Data))
			}
		}
	}
}

func (e *Env) Close() error {
	_, err := e.sendCommand(procCommand{Code: cmdShutdown})
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
	if err2 := e.stdW.Close(); err == nil {
		err = err2
	}
	if err2 := e.stdR.Close(); err == nil {
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
