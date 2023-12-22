package main

import (
	"amah/client/auth"
	"amah/client/monitor"
	"amah/service"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strings"
)

var scanMode = flag.Bool("scanMode", false, "enable scan mode")
var exeSuffix = flag.String("exeSuffix", "java", "match suffix of target application executable")
var normalMode = flag.Bool("normalMode", true, "enable normal mode that works as gateway and keeper")

var newUsername = flag.String("newUsername", "", "the new username to generate shadow line to append")
var newPassword = flag.String("newPassword", "", "the new password to generate shadow line to append")

func main() {
	flag.Parse()

	if *newUsername != "" && *newPassword != "" {
		line, err := auth.Register(*newUsername, *newPassword)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(line)
		return
	}

	if *normalMode {
		shadowLine, err := auth.Register("this_is_username", "this_is_password")
		if err != nil {
			log.Fatal(err)
		}
		accounts, _ := auth.ParseShadow(strings.NewReader(shadowLine))
		client, err := auth.NewClient(accounts)
		if err != nil {
			log.Fatal(err)
		}
		c := service.New(client, monitor.NewClient())
		err = http.ListenAndServe("0.0.0.0:8080", c)
		log.Fatal(err)
		return
	}

	processes, err := monitor.NewClient().Scan()
	if err != nil {
		log.Fatal(err)
	}
	if *scanMode {
		for _, app := range processes {
			fmt.Printf("%+v\n", app)
		}
		return
	}
	targets := filterByExecutableSuffix(processes, *exeSuffix)
	if len(targets) > 1 {
		slog.Warn("multiple result", "count", len(targets), "suffix", *exeSuffix)
	}
	for _, target := range targets {
		fmt.Println(target)
	}
	if len(targets) == 0 {
		proc, err := monitor.NewProcess(targets[0].PID)
		if err != nil {
			slog.Error(err.Error())
			return
		}
		fmt.Println(proc)
	}
}

func filterByExecutableSuffix(apps []monitor.Process, suffix string) []monitor.Process {
	var ret []monitor.Process
	for _, app := range apps {
		if strings.HasSuffix(app.Path, suffix) {
			ret = append(ret, app)
		}
	}
	return ret
}
