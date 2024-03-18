package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/comp529/qrlnc"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensures the server goroutine is terminated.

	go qrlnc.Server(ctx)
	time.Sleep(1 * time.Second) // Wait for the server to initialize.

	qrlnc.Client("test.m4s")

	// wait for the server to finish
	time.Sleep(2 * time.Second)

	original, err := ioutil.ReadFile("test.m4s")
	if err != nil {
		fmt.Printf("Error opening original file: %v", err)
	}

	// combine all the chunks
	recvfile := []byte{}
	for i := 0; i <= len(original)/qrlnc.CHUNKSIZE; i++ {
		filename := fmt.Sprintf("recv_%d.m4s", i)
		chunk, err := ioutil.ReadFile(filename)
		if err != nil {
			fmt.Printf("Error opening received file: %v\n", err)
			return
		}
		recvfile = append(recvfile, chunk...)
	}

	if err := os.WriteFile("recv.m4s", recvfile, 0644); err != nil {
		fmt.Printf("Failed to save file: %v\n", err)
		return
	}

	fmt.Printf("## Original file size: %d bytes\n", len(original))
	fmt.Printf("## Received file size: %d bytes\n", len(recvfile))

	if !bytes.Equal(original, recvfile[:len(original)]) {
		fmt.Printf("## Files do not match.\n")
	} else {
		fmt.Printf("## Successfully decoded all packets at the receiver.\n")
	}
}
