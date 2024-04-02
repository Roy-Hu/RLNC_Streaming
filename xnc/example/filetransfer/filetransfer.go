package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/comp529/xnc"
)

func main() {
	rootDir := "/var/www/html/tos_4sec_full/4K_dataset/4_sec/x264/bbb/DASH_Files/full/"
	filepath := filepath.Join(rootDir, "test.m4s")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensures the server goroutine is terminated.

	go xnc.Server(ctx, rootDir)
	time.Sleep(1 * time.Second) // Wait for the server to initialize.

	recvfile, _, _ := xnc.Client("test.m4s", true)

	// wait for the server to finish
	time.Sleep(2 * time.Second)

	original, err := ioutil.ReadFile(filepath)
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
