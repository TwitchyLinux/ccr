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
)

type procCommand struct {
	Code cmdCode

	Args []string
}

type procResp struct {
	Code  cmdCode
	Error string
}

func (e *Env) sendCommand(cmd procCommand) error {
	e.cmdW.SetWriteDeadline(time.Now().Add(cmdTimeout))
	if err := e.enc.Encode(cmd); err != nil {
		return err
	}

	e.respR.SetReadDeadline(time.Now().Add(cmdTimeout))
	var resp procResp
	if err := e.dec.Decode(&resp); err != nil {
		return err
	}
	if resp.Code != cmd.Code {
		return fmt.Errorf("bad response: code %v != %v", resp.Code, cmd.Code)
	}
	if resp.Error != "" {
		return errors.New(resp.Error)
	}
	return nil
}
