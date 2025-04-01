package service

import (
	"amah/client/application"
	"amah/client/monitor"
	"log/slog"
	"os"
	"path"
	"strings"
)

type Node struct {
	Process  monitor.Process
	Children []*Node `json:",omitempty"`
}

type ApplicationComplex struct {
	application.Application
	Instances []*Node `json:",omitempty"`
}

func CombineTheoryAndReality(applications []application.Application, processes []monitor.Process) []ApplicationComplex {
	ppidToProcesses := make(map[int][]monitor.Process)
	for _, process := range processes {
		ppidToProcesses[process.PPID] = append(ppidToProcesses[process.PPID], process)
	}

	appIDToRoots := combine(applications, processes)
	// Yes, there is a one line alternative as
	// fulfillChildrenRecursively(slices.Concat(slices.Collect(maps.Values(appIDToRoots))...), ppidToProcesses)
	// I just don't find it more readable or have any performance advantages.
	for _, roots := range appIDToRoots {
		fulfillChildrenRecursively(roots, ppidToProcesses)
	}

	var ret []ApplicationComplex
	for _, app := range applications {
		ret = append(ret, ApplicationComplex{
			Application: app,
			Instances:   appIDToRoots[app.ID],
		})
	}
	return ret
}

// fulfillChildrenRecursively fill children of each Node on every depth with data in ppidToProcesses.
// Modification is done in place on nodes, thereafter return value is not used. Such design for shared Node.
func fulfillChildrenRecursively(nodes []*Node, ppidToProcesses map[int][]monitor.Process) {
	queue := nodes
	for len(queue) > 0 {
		var next []*Node
		for _, parent := range queue {
			var children []*Node
			for _, child := range ppidToProcesses[parent.Process.PID] {
				children = append(children, &Node{
					Process:  child,
					Children: nil,
				})
			}
			parent.Children = children

			if len(children) > 0 {
				next = append(next, children...)
			}
		}
		queue = next
	}
}

func combine(
	applications []application.Application,
	processes []monitor.Process,
) (appIDToRoots map[int][]*Node) {
	appIDToRoots = make(map[int][]*Node)
	for _, proc := range processes {
		for _, app := range applications {
			if Similar(app, proc) {
				appIDToRoots[app.ID] = append(appIDToRoots[app.ID], &Node{
					Process:  proc,
					Children: nil,
				})
			}
		}
	}
	return appIDToRoots
}

func Similar(a application.Application, p monitor.Process) bool {
	if path.Base(a.Exec.Path) != path.Base(p.Path) {
		return false
	}
	aStat, err := os.Stat(a.AbsolutePath())
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Warn("compare app and proc: ignore as app bad stat", "app", a, "err", err)
		}
		return false
	}
	pStat, err := os.Stat(p.Path)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Warn("compare app and proc: ignore as proc bad stat", "proc", p, "err", err)
		}
		return false
	}
	if !os.SameFile(aStat, pStat) {
		return false
	}
	return strings.Join(a.Exec.Args, " ") == strings.Join(p.Args[1:], " ")
}
