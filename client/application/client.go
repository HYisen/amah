package application

import (
	"amah/ring"
	"bufio"
	"context"
	"io"
	"log"
	"log/slog"
	"os"
	"os/exec"
)

type Client struct {
	appID   int
	buf     ring.Ring[string]
	query   chan chan []string
	cancel  context.CancelFunc
	process *os.Process
}

func NewClient(app Application, outputHistoryLength int) (*Client, error) {
	ret := &Client{
		appID:  app.ID,
		buf:    ring.New[string](outputHistoryLength),
		query:  make(chan chan []string),
		cancel: nil,
	}
	return ret, ret.start(app)
}

// tee pipes lines in ch to wc, and save it in Ring.
// While piping, also make it ready for query on data stored in Ring.
// In the end, would close wc. Runs forever until ctx is Done.
func (c *Client) tee(ctx context.Context, ch <-chan string, wc io.WriteCloser) {
	defer func(c io.Closer) {
		if err := c.Close(); err != nil {
			// Nothing else can I do here, just print a WARN.
			slog.Warn("tee close wc", "err", err)
		}
	}(wc)

	for {
		select {
		case line := <-ch:
			c.buf.Add(line)
			if _, err := wc.Write([]byte(line + "\n")); err != nil {
				slog.Error("tee output drop", "appID", c.appID, "err", err, "line", line)
				log.Fatal(err) // Eager here as I'm not sure whether running without tee piping is acceptable.
			}
		case resp := <-c.query:
			resp <- c.buf.Get()
			// ref https://stackoverflow.com/questions/8593645/is-it-ok-to-leave-a-channel-open
			// I don't have to close it, just confirm it's a one-shot round-trip,
			// prevent it from waiting for more response forever.
			close(resp)
		case <-ctx.Done():
			return
		}
	}
}

// PID returns that of the running process. Returns zero if it has stopped properly. ref Terminate
func (c *Client) PID() int {
	if c.process == nil {
		return 0
	}
	return c.process.Pid
}

// start starts an app. One shall not execute start for multiple times.
func (c *Client) start(a Application) error {
	// Using CommandContext and cancel the ctx is identical in underlying,
	// comparing to Process.Kill. Since I also need PID, the latter is chosen.
	cmd := exec.Command(a.Exec.Path, a.Exec.Args...)
	cmd.Dir = a.Exec.WorkingDirectory

	cout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cerr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}
	c.process = cmd.Process

	ch := make(chan string)

	go func(dst chan<- string, src io.ReadCloser) {
		scanner := bufio.NewScanner(src)
		for scanner.Scan() {
			dst <- scanner.Text()
		}
	}(ch, cout)
	go func(dst chan<- string, src io.ReadCloser) {
		scanner := bufio.NewScanner(src)
		for scanner.Scan() {
			dst <- "!" + scanner.Text() // I just like it, comparing to use less stable bold red style.
		}
	}(ch, cerr)

	fp, err := os.Create(a.AbsoluteRedirectPath())
	if err != nil {
		return err
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	c.cancel = cancelFunc
	go c.tee(ctx, ch, fp)
	return nil
}

func (c *Client) Query() []string {
	if c.cancel == nil {
		return nil
	}
	ch := make(chan []string)
	c.query <- ch
	return <-ch
}

// Terminate kills the process, and then stop its helper.
// Terminate for multiple times will cause NPE, and panics as close a channel for more than one time.
// I design such kind of dangerous behaviour to prevent defensive calls of Terminate.
func (c *Client) Terminate() error {
	if err := c.process.Kill(); err != nil {
		return err
	}
	if _, err := c.process.Wait(); err != nil {
		return err
	}

	c.process = nil
	c.cancel()
	c.cancel = nil
	close(c.query)
	return nil
}
