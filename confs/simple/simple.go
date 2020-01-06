package simple

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Mode uint8

// Valid modes.
const (
	ModeLines Mode = iota
	ModeKV
)

type Config struct {
	Mode Mode

	Disallowed   []string
	SkipComments bool
}

type ParsedConf struct {
	KV    map[string]string
	Lines []string
}

func Parse(c Config, r io.Reader) (*ParsedConf, error) {
	var (
		s   = bufio.NewScanner(r)
		out = ParsedConf{}
		err error
		i   int
	)

	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			i++
			continue
		}
		if strings.HasPrefix(line, "#") && c.SkipComments {
			i++
			continue
		}
		for _, c := range c.Disallowed {
			if i := strings.Index(line, c); i >= 0 {
				return nil, fmt.Errorf("line %d: illegal sequence %q", i+1, c)
			}
		}

		switch c.Mode {
		case ModeKV:
			idx := strings.Index(line, "=")
			if idx <= 0 {
				return nil, fmt.Errorf("line %d: no key in key=value entry", i+1)
			}
			if idx+1 >= len(line) {
				return nil, fmt.Errorf("line %d: no value for key %q", i+1, line[:idx])
			}
			key, val := strings.TrimSpace(line[:idx]), strings.TrimSpace(line[idx+1:])
			if strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"") {
				if val, err = strconv.Unquote(val); err != nil {
					return nil, fmt.Errorf("line %d: bad quotation: %v", i+1, err)
				}
			}
			out.KV[key] = val

		case ModeLines:
			out.Lines = append(out.Lines, line)
		default:
			return nil, fmt.Errorf("unrecognized parser mode: %v", c.Mode)
		}

		i++
	}
	if err := s.Err(); err != nil {
		return nil, fmt.Errorf("scanning: %v", err)
	}
	return &out, nil
}
