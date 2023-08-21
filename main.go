package main

import (
	"amah/monitor"
	"fmt"
	"log"
)

func main() {
	apps, err := monitor.Scan()
	if err != nil {
		log.Fatal(err)
	}
	for _, app := range apps {
		fmt.Printf("%+v\n", app)
	}
}
