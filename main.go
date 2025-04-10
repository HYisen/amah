package main

import (
	"amah/client/application"
	"amah/client/auth"
	"amah/client/monitor"
	"amah/proxy"
	"amah/service"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

var scanMode = flag.Bool("scanMode", false, "enable scan mode")
var exeSuffix = flag.String("exeSuffix", "java", "match suffix of target application executable")
var normalMode = flag.Bool("normalMode", true, "enable normal mode that works as gateway and keeper")
var appConfigPath = flag.String("appConfigPath", "apps.yaml", "the applications config path")
var shadowPath = flag.String("shadowPath", "shadow", "where the shadow file exist")

var listenAddress = flag.String("listenAddress", "0.0.0.0:8080", "where the server serve")
var certFile = flag.String("certFile", "", "HTTPS cert filepath, not empty no HTTP")
var keyFile = flag.String("keyFile", "", "HTTPS key filepath, not empty no HTTP")

var portBasic = flag.Int("portBasic", 8600, "where the control plane serve on localhost")

var newUsername = flag.String("newUsername", "", "the new username to generate shadow line to append")
var newPassword = flag.String("newPassword", "", "the new password to generate shadow line to append")

type GraveyardKeeper struct {
	servers []*http.Server
}

func (gk *GraveyardKeeper) Add(server *http.Server) {
	gk.servers = append(gk.servers, server)
}

func (gk *GraveyardKeeper) Shutdown(ctx context.Context) error {
	var ret error
	for _, server := range gk.servers {
		// Best effort rather than fail fast, overdose is better than skip in shutdown.
		ret = errors.Join(server.Shutdown(ctx))
	}
	return ret
}

func LogFatalUnexpectedError(err error) {
	if !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

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
		gk := &GraveyardKeeper{}

		accounts, _ := auth.LoadAccounts(*shadowPath)
		// Ignore err, init an empty accounts if no shadow file
		client, err := auth.NewClient(accounts)
		if err != nil {
			log.Fatal(err)
		}
		repository, err := application.NewRepository(*appConfigPath)
		if err != nil {
			log.Fatal(err)
		}
		c := service.New(client, monitor.NewClient(), repository, gk)
		basicAddr := "localhost:" + strconv.Itoa(*portBasic)
		basicServer := &http.Server{Addr: basicAddr, Handler: c}
		gk.Add(basicServer)
		slog.Info("starting basic", "addr", basicAddr)
		go func() {
			LogFatalUnexpectedError(basicServer.ListenAndServe())
		}()

		cfg, err := proxy.NewConfig("router.yaml")
		if err != nil {
			log.Fatal(err)
		}
		p := proxy.New(cfg, client)
		server := &http.Server{Addr: *listenAddress, Handler: p}
		gk.Add(server)
		slog.Info("starting gateway", "addr", *listenAddress)
		if *certFile == "" && *keyFile == "" {
			LogFatalUnexpectedError(server.ListenAndServe())
		} else {
			LogFatalUnexpectedError(server.ListenAndServeTLS(*certFile, *keyFile))
		}
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
