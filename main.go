package main

import (
	"amah/client/application"
	"amah/client/auth"
	"amah/client/monitor"
	"amah/service"
	"bytes"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
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
var addrOther = flag.String("addrOther", "https://localhost:8443", "where the fallback serve")

var newUsername = flag.String("newUsername", "", "the new username to generate shadow line to append")
var newPassword = flag.String("newPassword", "", "the new password to generate shadow line to append")

func NewProxy(basic *url.URL, other *url.URL) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetXForwarded()
			// I have searched it in Eta0, the v1 prefix algorithm shall work. Expand it if this becomes more complex.
			if strings.HasPrefix(r.In.URL.Path, "/v1") {
				r.SetURL(basic)
			} else {
				r.SetURL(other)
			}
		},
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
		file, err := os.ReadFile(*shadowPath)
		if err != nil {
			log.Fatal(err)
		}
		accounts, _ := auth.ParseShadow(bytes.NewReader(file))
		client, err := auth.NewClient(accounts)
		if err != nil {
			log.Fatal(err)
		}
		repository, err := application.NewRepository(*appConfigPath)
		if err != nil {
			log.Fatal(err)
		}
		c := service.New(client, monitor.NewClient(), repository)
		// localhost so HTTP is acceptable
		basic, err := url.Parse(fmt.Sprintf("http://localhost:%d", *portBasic))
		if err != nil {
			log.Fatal(err)
		}
		go func() {
			err = http.ListenAndServe(basic.Host, c)
			log.Fatal(err)
		}()

		other, err := url.Parse(*addrOther)
		if err != nil {
			log.Fatal(err)
		}
		p := NewProxy(basic, other)
		log.Printf("listen on %s\n", *listenAddress)
		if *certFile == "" && *keyFile == "" {
			if err = http.ListenAndServe(*listenAddress, p); err != nil {
				log.Fatal(err)
			}
		} else {
			if err = http.ListenAndServeTLS(*listenAddress, *certFile, *keyFile, p); err != nil {
				log.Fatal(err)
			}
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
