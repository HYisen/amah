package monitor

import (
	"errors"
	"fmt"
	"github.com/prometheus/procfs"
	"os/exec"
	"strconv"
	"strings"
)

type Client struct {
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) Scan() ([]Process, error) {
	procs, err := procfs.AllProcs()
	if err != nil {
		return nil, err
	}
	var ret []Process
	for _, proc := range procs {
		valid, app, err := newProcess(proc)
		if err != nil {
			return nil, err
		}
		if !valid {
			continue
		}
		ret = append(ret, app)
	}
	return ret, err
}

// Kill kills the process by PID, if no such PID, would return false found and nil err.
func (c *Client) Kill(PID int) (found bool, err error) {
	if _, err = exec.Command("kill", strconv.Itoa(PID)).Output(); err != nil {
		var e *exec.ExitError
		if errors.As(err, &e) {
			msg := string(e.Stderr)
			if strings.HasSuffix(msg, " failed: No such process\n") {
				return false, nil
			}
			return false, fmt.Errorf("stderr[%v]: %v", msg, e)
		} else {
			return false, err
		}
	}
	return true, nil
}
