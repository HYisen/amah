package monitor

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/prometheus/procfs"
	"strings"
)

type Application struct {
	Path string
	Args []string
	PID  int    // Process ID
	PPID int    // Parent Process ID
	RSS  uint64 // Resident Set Size, a memory usage metric of how much needed.
	PSS  uint64 // Proportional Set Size, a memory usage metric of how much used.
}

func (a Application) String() string {
	return fmt.Sprintf(
		"%d(%d)\t%s\t%s\t%s\t%v",
		a.PID,
		a.PPID,
		a.Path,
		humanize.IBytes(a.RSS),
		humanize.IBytes(a.PSS),
		a.Args,
	)
}

func NewApplication(pid int) (Application, error) {
	proc, err := procfs.NewProc(pid)
	if err != nil {
		return Application{}, err
	}
	valid, app, err := newApplication(proc)
	if err != nil {
		return Application{}, err
	}
	if !valid {
		return Application{}, fmt.Errorf("invalid app on PID %d", pid)
	}
	return app, nil
}

// newApplication creates Application from its Proc, if not valid as common not privilege as root or no such process,
// valid would be false and the app shall be ignored. If exceptional failure, err not nil and shall panic.
func newApplication(proc procfs.Proc) (valid bool, app Application, err error) {
	executable, err := proc.Executable()
	if err != nil {
		// If not run as root, only runner user's processes are visible. Common and keep silent.
		if strings.HasSuffix(err.Error(), "permission denied") {
			return false, Application{}, nil
		}
		return false, Application{}, err
	}
	stat, err := proc.Stat()
	if err != nil {
		return false, Application{}, err
	}
	args, err := proc.CmdLine()
	if err != nil {
		return false, Application{}, err
	}
	rollup, err := proc.ProcSMapsRollup()
	if err != nil {
		// Some not normal applications like [kthreadd] just end with no such process on the smaps_rollup file,
		// usually they are not our targets, just silent ignore.
		if strings.HasSuffix(err.Error(), "no such process") {
			return false, Application{}, nil
		}
		return false, Application{}, err
	}
	return true, Application{
		Path: executable,
		Args: args,
		PID:  stat.PID,
		PPID: stat.PPID,
		RSS:  rollup.Rss,
		PSS:  rollup.Pss,
	}, nil
}

func Scan() ([]Application, error) {
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
