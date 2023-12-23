package application

import (
	"amah/ring"
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/exec"
)

type Client struct {
	appID  int
	buf    ring.Ring[string]
	query  chan chan []string
	cancel context.CancelFunc
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

func (c *Client) start(a Application) error {
	if c.cancel != nil {
		return fmt.Errorf("unhandled cancel")
	}

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

// Terminate stops the running of helper, which little relevant to the started app.
// It means the tee mechanism stops pip output to RedirectPath and the Query is no longer available.
func (c *Client) Terminate() {
	c.cancel()
	c.cancel = nil
	close(c.query)
}
