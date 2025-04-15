package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var num = flag.Int("num", 5, "the count down seconds")

var stub = flag.Bool("stub", false, "whether the program would ignore SIGINT")

func main() {
	flag.Parse()
	if *stub {
		signal.Ignore(syscall.SIGINT)
	}

	fmt.Println("PID", os.Getpid())

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGKILL, syscall.SIGINT, syscall.SIGTERM)
	go printSignal(ch)

	for ttl := *num; ttl > 0; ttl-- {
		fmt.Println(ttl)
		time.Sleep(time.Second)
	}
	_, _ = fmt.Fprintln(os.Stderr, "BOOM!")
}

func printSignal(ch <-chan os.Signal) {
	for sig := range ch {
		log.Printf("received signal %s", sig.String())
	}
}
