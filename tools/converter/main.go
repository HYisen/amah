package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"log"
	"os"
	"strings"
)

func main() {
	input, err := os.ReadFile("input.txt")
	if err != nil {
		log.Fatal(input)
	}

	output, err := os.Create("output.csv")
	if err != nil {
		log.Fatal(err)
	}
	writer := csv.NewWriter(output)
	defer writer.Flush()
	if err := writer.Write([]string{"speed ns/op", "fields"}); err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(input))
	for scanner.Scan() {
		one := scanner.Text()
		// BenchmarkRing_Get/v0_1k_25%
		fields := strings.Split(one[strings.LastIndexByte(one, '/')+1:], "_")
		scanner.Scan()
		two := scanner.Text()
		// BenchmarkRing_Get/v0_1k_25%-20         	 1708902	       653.0 ns/op
		two, _ = strings.CutSuffix(two, " ns/op")
		speed := two[strings.LastIndexByte(two, ' ')+1:]
		if err := writer.Write(append([]string{speed}, fields...)); err != nil {
			log.Fatal(err)
		}
	}
}
