package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"time"
)

func solve(c string) {
	suffix, err := base64.StdEncoding.DecodeString(c)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	fmt.Printf("Suffix: %x\n", suffix)
	t := time.Now()
	for {
		solution := make([]byte, 10)
		rand.Read(solution)
		sum := sha512.Sum512(solution)
		if bytes.HasPrefix(sum[:], suffix) {
			pow := append(suffix, solution...)
			fmt.Printf("PoW: %s\n", base64.StdEncoding.EncodeToString(pow))
			break
		}
	}
	fmt.Println("Time", time.Since(t).Seconds())
}

func pow(c string) string {
	suffix, err := base64.StdEncoding.DecodeString(c)
	if err != nil {
		return ""
	}
	for {
		solution := make([]byte, 10)
		rand.Read(solution)
		sum := sha512.Sum512(solution)
		if bytes.HasPrefix(sum[:], suffix) {
			pow := append(suffix, solution...)
			solution := base64.StdEncoding.EncodeToString(pow)
			return solution
		}
	}
	return ""
}

func main() {
	challenge := flag.String("c", "", "")
	flag.Parse()
	if *challenge == "" {
		fmt.Println("empty challange")
		os.Exit(1)
	}
	solve(*challenge)
}
