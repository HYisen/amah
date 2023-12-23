package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

var num = flag.Int("num", 5, "the count down seconds")

func main() {
	flag.Parse()
	for ttl := *num; ttl > 0; ttl-- {
		fmt.Println(ttl)
		time.Sleep(time.Second)
	}
	_, _ = fmt.Fprintln(os.Stderr, "BOOM!")
}
