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

	"time"

	"github.com/itzmeanjan/kodr"
	"github.com/itzmeanjan/kodr/full"
	"github.com/quic-go/quic-go"
)

func Server(ctx context.Context, rootDir string) {

	quicConf := &quic.Config{
		EnableDatagrams: true,
	}

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
			ctx := context.Background()
			sess, err := listener.Accept(ctx)
			if err != nil {
				fmt.Println("[Server] Failed to accept session:", err)
				return
			}

			go handleSession(sess, rootDir)
		}
	}
}

func handleSession(sess quic.Connection, rootDir string) {
	// After file is fully received
	fmt.Println("[Server] Session started, waiting for file transfers...")
	ctx := context.Background()

	// for {
	// Accept a new stream within the session.
	// stream, err := sess.AcceptStream(ctx) // Using context.Background() for simplicity; adjust as needed.
	// if err != nil {
	// 	if err == io.EOF {
	// 		// The session was closed gracefully.
	// 		fmt.Println("[Server] Session closed by client.")
	// 		return
	// 	}
	// 	fmt.Printf("[Server] Error accepting stream: %v\n", err)
	// 	return // Or continue to try accepting new streams, depending on your error handling strategy.
	// }

	// go func() {
	fmt.Println("[Server] Stream accepted, waiting for init packet...")

	accu_recv := 0
	buffer := make([]byte, 0, INITSIZE)
	for accu_recv < INITSIZE {
		msg, err := sess.ReceiveMessage(ctx)
		n := len(msg)
		buffer = append(buffer, msg...)
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
		sendFile(sess, filepath, true)
	} else {
		sendFile(sess, filepath, false)
	}
	// }()
	// }
}

func sendFile(sess quic.Connection, filename string, encode bool) {
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
	var pieces uint
	if encode {
		pieces = CODEDPIECECNT
	} else {
		pieces = PIECECNT
	}

	for i := 0; i < len(chunks); i++ {
		if i == len(chunks)-1 {
			size = len(filebytes) - CHUNKSIZE*(len(chunks)-1)
		} else {
			size = CHUNKSIZE
		}

		fmt.Printf("[Server] Sending chunk %v, %v pieces\n", i, pieces)
		// hasher := sha512.New512_224()
		// hasher.Write(chunks[i])

		codedPieces := make([]*kodr.CodedPiece, 0, CODEDPIECECNT)

		if encode {
			enc, err := full.NewFullRLNCEncoderWithPieceCount(chunks[i], PIECECNT)
			if err != nil {
				log.Printf("Error: %s\n", err.Error())
				return
			}

			for j := 0; j < int(CODEDPIECECNT); j++ {
				codedPieces = append(codedPieces, enc.CodedPiece())
			}
		}

		for s := 0; s < int(pieces); s++ {
			var pktE []byte

			if encode {
				pktE, err = GetXNCEncPkt(size, i, len(chunks), codedPieces[s])
			} else {
				pktE, err = GetXNCPkt(size, i, len(chunks), chunks[i][s*PIECESIZE:(s+1)*PIECESIZE])
			}

			if err != nil {
				fmt.Printf("Error encoding packet data: %v", err)
				return
			}

			err = sess.SendMessage(pktE)
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
			// loss debug
			fmt.Printf("[Server] Chunk %d, sent %d\n", i, s)
		}
	}

	for i := 0; i < 5; i++ {
		endpkt := EncodeEND(len(chunks)-1, encode)
		err = sess.SendMessage(endpkt)
		time.Sleep(5 * time.Millisecond)
	}

	fmt.Printf("[Server] Finished sending file\n")
}
