package main

import (
	"context"
	"fmt"

	"github.com/comp529/xnc"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensures the server goroutine is terminated.

	xnc.Server(ctx, xnc.RootDir)

	fmt.Println("Server End")
}
