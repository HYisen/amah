package main

import (
	"amah/controller"
	"amah/monitor"
	"amah/service/auth"
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
		service, err := auth.NewService(accounts)
		if err != nil {
			log.Fatal(err)
		}
		c := controller.New(service)
		err = http.ListenAndServe("0.0.0.0:8080", c)
		log.Fatal(err)
		return
	}

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
