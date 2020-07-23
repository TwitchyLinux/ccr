package proc

import (
	"bytes"
	"encoding/gob"
  "os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"

	"github.com/docker/docker/pkg/reexec"
)

type outputData struct {
	ProcID   string
	IsStderr bool
	Data     []byte

	Complete bool
	ExitCode int
	Error    string
}

func runBlocking(cmd procCommand, pivotDir string, readOnly bool) procResp {
	c := reexec.Command(append([]string{"reexecEntry", "run", pivotDir, strconv.FormatBool(readOnly), cmd.Dir}, cmd.Args...)...)
	var sOut, sErr bytes.Buffer
	c.Stdout = &sOut
	c.Stderr = &sErr
	c.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS,
	}
	resp := procResp{Code: cmd.Code}
	if err := c.Run(); err != nil {
		resp.Error = err.Error()
		if eErr, isExecErr := err.(*exec.ExitError); isExecErr {
			resp.ExitCode = eErr.ExitCode()
		}
	}
	resp.Stderr = sErr.Bytes()
	resp.Stdout = sOut.Bytes()
	return resp
}

type execManager struct {
	out       *gob.Encoder
	processes map[string]*exec.Cmd
	stream    chan outputData

	l       sync.Mutex
	wg      sync.WaitGroup
	closed bool
}

func (m *execManager) Close() error {
	if m.closed {
		return nil
	}
  m.closed = true

  m.l.Lock()
  for _, c := range m.processes {
    c.Process.Kill()
  }
  m.l.Unlock()
	m.wg.Wait() // Wait for all process watchers (the Wait()'ers) to finish.
  close(m.stream)
	return nil
}

func (m *execManager) outputStreamer() {
	for  msg := range m.stream {
    m.out.Encode(msg)
	}
}

type streamWriter struct {
	id string
  isErr bool

	m  *execManager
}

func (w *streamWriter) Write(b []byte) (int, error) {
  if w.m.closed {
    return 0, os.ErrClosed
  }
  w.m.stream <- outputData{Data: b, ProcID: w.id, IsStderr: w.isErr}
	return len(b), nil
}

func (m *execManager) watchProc(c *exec.Cmd, id string) {
  defer m.wg.Done()
  defer func(){
    m.l.Lock()
    defer m.l.Unlock()
    delete(m.processes, id)
  }()

  od := outputData{ProcID: id,Complete: true}
  if err := c.Wait(); err != nil {
    od.Error = err.Error()
    if eErr, isExecErr := err.(*exec.ExitError); isExecErr {
      od.ExitCode = eErr.ExitCode()
    }
  }
  m.stream <- od
}

func (m *execManager) RunStreaming(cmd procCommand, pivotDir string, readOnly bool) procResp {
	m.l.Lock()
	defer m.l.Unlock()
  if m.closed {
    return procResp{Code: cmd.Code, Error: os.ErrClosed.Error()}
  }

	c := reexec.Command(append([]string{"reexecEntry", "run", pivotDir, strconv.FormatBool(readOnly), cmd.Dir}, cmd.Args...)...)
	c.Stdout = &streamWriter{m: m, id: cmd.ProcID, isErr: false}
	c.Stderr = &streamWriter{m: m, id: cmd.ProcID, isErr: true}
	c.SysProcAttr = &syscall.SysProcAttr{Cloneflags: syscall.CLONE_NEWNS}
	resp := procResp{Code: cmd.Code}

	if err := c.Start(); err != nil {
		resp.Error = err.Error()
		if eErr, isExecErr := err.(*exec.ExitError); isExecErr {
			resp.ExitCode = eErr.ExitCode()
		}
	} else {
    m.wg.Add(1)
    m.processes[cmd.ProcID] = c
    go m.watchProc(c, cmd.ProcID)
  }
	return resp
}

func makeExecManager(out *gob.Encoder) (*execManager, error) {
	m := execManager{
		out:       out,
		processes: map[string]*exec.Cmd{},
		stream:    make(chan outputData),
	}
	go m.outputStreamer()
	return &m, nil
}
