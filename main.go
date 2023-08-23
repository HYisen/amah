package main

import (
	"amah/monitor"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"strings"
)

var scanMode = flag.Bool("scanMode", false, "enable scan mode")
var exeSuffix = flag.String("exeSuffix", "java", "match suffix of target application executable")

func main() {
	flag.Parse()

	apps, err := monitor.Scan()
	if err != nil {
		log.Fatal(err)
	}
	if *scanMode {
		for _, app := range apps {
			fmt.Printf("%+v\n", app)
		}
		return
	}
	targets := filterByExecutableSuffix(apps, *exeSuffix)
	if len(targets) > 1 {
		slog.Warn("multiple result", "count", len(targets), "suffix", *exeSuffix)
	}
	for _, target := range targets {
		fmt.Println(target)
	}
	if len(targets) == 0 {
		app, err := monitor.NewApplication(targets[0].PID)
		if err != nil {
			slog.Error(err.Error())
			return
		}
		fmt.Println(app)
	}
}

func filterByExecutableSuffix(apps []monitor.Application, suffix string) []monitor.Application {
	var ret []monitor.Application
	for _, app := range apps {
		if strings.HasSuffix(app.Path, suffix) {
			ret = append(ret, app)
		}
	}
	return ret
}
