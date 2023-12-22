package monitor

import (
	"github.com/prometheus/procfs"
)

type Client struct {
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) Scan() ([]Application, error) {
	procs, err := procfs.AllProcs()
	if err != nil {
		return nil, err
	}
	var ret []Application
	for _, proc := range procs {
		valid, app, err := newApplication(proc)
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
