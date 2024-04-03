package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/comp529/xnc"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensures the server goroutine is terminated.

	go xnc.Server(ctx, xnc.RootDir)
	time.Sleep(1 * time.Second) // Wait for the server to initialize.

	recvfile, _, _ := xnc.Client(xnc.TestFile, true)

	// wait for the server to finish
	time.Sleep(2 * time.Second)

	original, err := ioutil.ReadFile(xnc.RootDir + xnc.TestFile)
	if err != nil {
		fmt.Printf("Error opening original file: %v", err)
	}

	fmt.Printf("## Original file size: %d bytes\n", len(original))
	fmt.Printf("## Received file size: %d bytes\n", len(recvfile))

	if !bytes.Equal(original, recvfile) {
		fmt.Printf("## Files do not match.\n")
	} else {
		fmt.Printf("## Successfully decoded all packets at the receiver.\n")
	}
}
