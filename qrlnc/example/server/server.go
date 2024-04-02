package main

import (
	"context"
	"fmt"

	"github.com/comp529/qrlnc"
)

func main() {
	rootDir := "/var/www/html/tos_4sec_full/4K_dataset/4_sec/x264/bbb/DASH_Files/full/"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensures the server goroutine is terminated.

	qrlnc.Server(ctx, rootDir)

	fmt.Println("Server End")
}
