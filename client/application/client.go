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
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type Client struct {
	appID   int
	buf     ring.Ring[string]
	query   chan chan []string
	cancel  context.CancelFunc
	process *os.Process
	mu      *sync.Mutex // guard Terminate
}

func NewClient(app Application, outputHistoryLength int) (*Client, error) {
	ret := &Client{
		appID:   app.ID,
		buf:     ring.New[string](outputHistoryLength),
		query:   make(chan chan []string),
		cancel:  nil,
		process: nil,
		mu:      &sync.Mutex{},
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
	if c.Stopped() {
		// Once c.Stopped(), c.buf becomes always safe to read only.
		return c.buf.Get()
	}
	ch := make(chan []string)
	c.query <- ch
	return <-ch
}

func (c *Client) Stopped() bool {
	// Because it's the final set in Terminate procedure.
	return c.cancel == nil
}

// Terminate kills the process, and then stop its helper.
// It's thread safe and okay to be invoked for multiple times.
func (c *Client) Terminate() error {
	// Why not TryLock? Because I don't want a raced one returns success earlier than the first one is done.
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Stopped() {
		return nil
	}

	// As long as c.cancel not nil, the c.process can not be nil.
	if err := c.process.Signal(syscall.SIGTERM); err != nil {
		return err
	}

	executed := &atomic.Bool{}
	// The 5s latency shall be long enough for normal terminations, and is not too long as an API timeout.
	go DelayExecution(5*time.Second, c.process, executed)

	if _, err := c.process.Wait(); err != nil {
		return err
	}
	executed.Store(true)

	c.process = nil
	c.cancel()
	c.cancel = nil
	close(c.query)
	return nil
}

func DelayExecution(latency time.Duration, process *os.Process, executed *atomic.Bool) {
	timer := time.NewTimer(latency)
	_ = <-timer.C
	if !executed.Load() {
		if err := process.Kill(); err != nil {
			// Yes, despite the usage of atomic, it still could race.
			// As long as the process is terminated but out Wait and set not completed yet,
			// which could be induced if you add a long time.Sleep right before executed.Store.
			// But it's usually safe, because even if the process pid is left unchanged after termination,
			// that pid is less likely to be reused within latency that shall be relatively short.
			// I just warn it, to help notify if the race does happen in production.
			slog.Warn("kill process", "pid", process.Pid, "err", err)
		}
		slog.Info("killed process as termination timeout", "pid", process.Pid)
	}
}
