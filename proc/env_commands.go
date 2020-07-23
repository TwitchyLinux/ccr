package proc

import (
	"errors"
	"fmt"
	"time"
)

type cmdCode uint8

const (
	cmdInvalid cmdCode = iota
	cmdShutdown
	cmdRunBlocking
	cmdRunStreaming
)

type procCommand struct {
	Code cmdCode

	Args []string
	Dir  string

	ProcID string
}

type procResp struct {
	Code  cmdCode
	Error string

	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

func (e *Env) sendCommand(cmd procCommand) (procResp, error) {
	e.cmdW.SetWriteDeadline(time.Now().Add(cmdTimeout))
	if err := e.enc.Encode(cmd); err != nil {
		return procResp{}, err
	}

	e.respR.SetReadDeadline(time.Now().Add(cmdTimeout))
	var resp procResp
	if err := e.dec.Decode(&resp); err != nil {
		return procResp{}, err
	}
	if resp.Code != cmd.Code {
		return procResp{}, fmt.Errorf("bad response: code %v != %v", resp.Code, cmd.Code)
	}
	if resp.Error != "" {
		return resp, errors.New(resp.Error)
	}
	return resp, nil
}
