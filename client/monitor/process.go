package monitor

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/prometheus/procfs"
	"strings"
)

type Process struct {
	Path string
	Args []string
	PID  int    // Process ID
	PPID int    // Parent Process ID
	RSS  uint64 // Resident Set Size, a memory usage metric of how much needed.
	PSS  uint64 // Proportional Set Size, a memory usage metric of how much used.
}

func (p Process) String() string {
	return fmt.Sprintf(
		"%d(%d)\t%s\t%s\t%s\t%v",
		p.PID,
		p.PPID,
		p.Path,
		humanize.IBytes(p.RSS),
		humanize.IBytes(p.PSS),
		p.Args,
	)
}

func NewProcess(pid int) (Process, error) {
	proc, err := procfs.NewProc(pid)
	if err != nil {
		return Process{}, err
	}
	valid, app, err := newProcess(proc)
	if err != nil {
		return Process{}, err
	}
	if !valid {
		return Process{}, fmt.Errorf("invalid app on PID %d", pid)
	}
	return app, nil
}

// newProcess creates Process from its Proc, if not valid as common not privilege as root or no such process,
// valid would be false and the app shall be ignored. If exceptional failure, err not nil and shall panic.
func newProcess(proc procfs.Proc) (valid bool, app Process, err error) {
	executable, err := proc.Executable()
	if err != nil {
		// If not run as root, only runner user's processes are visible. Common and keep silent.
		if strings.HasSuffix(err.Error(), "permission denied") {
			return false, Process{}, nil
		}
		return false, Process{}, err
	}
	stat, err := proc.Stat()
	if err != nil {
		return false, Process{}, err
	}
	args, err := proc.CmdLine()
	if err != nil {
		return false, Process{}, err
	}
	rollup, err := proc.ProcSMapsRollup()
	if err != nil {
		// Some not normal applications like [kthreadd] just end with no such process on the smaps_rollup file,
		// usually they are not our targets, just silent ignore.
		if strings.HasSuffix(err.Error(), "no such process") {
			return false, Process{}, nil
		}
		return false, Process{}, err
	}
	return true, Process{
		Path: executable,
		Args: args,
		PID:  stat.PID,
		PPID: stat.PPID,
		RSS:  rollup.Rss,
		PSS:  rollup.Pss,
	}, nil
}
