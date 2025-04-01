package monitor

import (
	"github.com/prometheus/procfs"
	"os"
	"syscall"
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
	// > On Unix systems, FindProcess always succeeds and returns a Process for the given pid,
	// At present I only test and guarantee user experience on Unix systems,
	// so we treat it as it is Unix and follow the guide in docs of os.FindProcess.
	process, _ := os.FindProcess(PID)
	if process.Signal(syscall.Signal(0)) != nil {
		return false, nil
	}
	return true, process.Kill()
}
