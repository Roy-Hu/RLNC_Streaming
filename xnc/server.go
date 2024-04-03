package xnc

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/itzmeanjan/kodr"
	"github.com/itzmeanjan/kodr/full"
	"github.com/lucas-clemente/quic-go"
)

func Server(ctx context.Context, rootDir string) {

	quicConf := &quic.Config{}
	tlsConf := GenerateTLSConfig()
	if tlsConf == nil {
		return
	}

	listener, err := quic.ListenAddr(clientaddr, tlsConf, quicConf)
	if err != nil {
		fmt.Println("[Server] Failed to start server:", err)
		return
	}
	defer listener.Close()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("[Server] Server shutting down")
			return
		default:
			sess, err := listener.Accept()
			if err != nil {
				fmt.Println("[Server] Failed to accept session:", err)
				return
			}

			go handleSession(sess, rootDir)
		}
	}
}

func handleSession(sess quic.Session, rootDir string) {
	// After file is fully received
	fmt.Println("[Server] Session started, waiting for file transfers...")

	for {
		// Accept a new stream within the session.
		stream, err := sess.AcceptStream() // Using context.Background() for simplicity; adjust as needed.
		if err != nil {
			if err == io.EOF {
				// The session was closed gracefully.
				fmt.Println("[Server] Session closed by client.")
				return
			}
			fmt.Printf("[Server] Error accepting stream: %v\n", err)
			return // Or continue to try accepting new streams, depending on your error handling strategy.
		}

		go func() {
			fmt.Println("[Server] Stream accepted, waiting for init packet...")

			accu_recv := 0
			buffer := make([]byte, INITSIZE)
			for accu_recv < INITSIZE {
				n, err := stream.Read(buffer[accu_recv:])
				if err != nil {
					if err == io.EOF {
						fmt.Printf("[Server] Stream closed by server")
						break
					}
					fmt.Println("[Server] Error reading from stream:", err)
					continue // or handle the error appropriately
				} else {
					accu_recv += n
				}
			}

			init, err := DecodeInit(buffer)
			if err != nil {
				fmt.Printf("[Server] Error decoding init packet: %v", err)
				return
			}
			fmt.Printf("[Server] Client request file: %v\n", init.Filename)
			filepath := filepath.Join(rootDir, init.Filename)

			if init.Type == TYPE_INIT_ENC {
				sendFile(stream, filepath, true)
			} else {
				sendFile(stream, filepath, false)
			}
		}()
	}
}

func sendFile(stream quic.Stream, filename string, encode bool) {
	rand.Seed(42)

	file, err := os.Open(filename)
	if err != nil {
		fmt.Errorf("[Server] Error opening file: %v", err)
		return
	}

	defer file.Close() // Ensure the file is closed after reading

	filebytes, err := io.ReadAll(file)
	if err != nil {
		fmt.Errorf("[Server] Error reading file: %v", err)
		return
	}

	fmt.Printf("[Server] Read %d bytes from %v\n", len(filebytes), filename)
	chunks := SpiltFile(filebytes, CHUNKSIZE)
	fmt.Printf("[Server] Split file into %v chunks\n", len(chunks))

	var size int
	var chunkId int

	for i := 0; i < len(chunks); i++ {
		if i == len(chunks)-1 {
			size = len(filebytes) - CHUNKSIZE*(len(chunks)-1)
			chunkId = END_CHUNK
		} else {
			size = CHUNKSIZE
			chunkId = i
		}

		fmt.Printf("[Server] Sending chunk %v of size %v\n", i, size)
		// hasher := sha512.New512_224()
		// hasher.Write(chunks[i])

		enc, err := full.NewFullRLNCEncoderWithPieceCount(chunks[i], PIECECNT)
		if err != nil {
			log.Printf("Error: %s\n", err.Error())
			return
		}

		codedPieces := make([]*kodr.CodedPiece, 0, CODEDPIECECNT)
		for j := 0; j < int(CODEDPIECECNT); j++ {
			codedPieces = append(codedPieces, enc.CodedPiece())
		}

		for s := 0; s < int(CODEDPIECECNT); s++ {
			pktE, err := GetXNCPkt(size, chunkId, codedPieces[s])
			if err != nil {
				fmt.Printf("Error encoding packet data: %v", err)
				return
			}

			_, err = stream.Write(pktE)
			if err != nil {
				if err == io.EOF {
					// The stream has been closed by the server, gracefully exit the loop.
					fmt.Printf("Stream closed by the client, stopping write operations.\n")
					break
				} else if strings.Contains(err.Error(), "closed stream") {
					fmt.Printf("Stream closed by the server, stopping write operations.\n")
					continue
				} else {
					// Handle other errors that might not necessitate stopping.
					fmt.Printf("Error writing to stream: %v\n", err)
					continue
				}
			}
		}
	}

	fmt.Printf("[Server] Finished sending file\n")
}
