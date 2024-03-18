package qrlnc

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/lucas-clemente/quic-go"
)

func Server(ctx context.Context) {
	quicConf := &quic.Config{}
	tlsConf := GenerateTLSConfig()
	if tlsConf == nil {
		return
	}

	listener, err := quic.ListenAddr("localhost:4242", tlsConf, quicConf)
	if err != nil {
		fmt.Println("Failed to start server:", err)
		return
	}
	defer listener.Close()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Server shutting down")
			return
		default:
			sess, err := listener.Accept()
			if err != nil {
				fmt.Println("Failed to accept session:", err)
				return
			}

			go handleSession(sess)
		}
	}
}

func handleSession(sess quic.Session) {
	// After file is fully received
	fmt.Println("Session started, waiting for file transfers...")

	for {
		// Accept a new stream within the session.
		stream, err := sess.AcceptStream() // Using context.Background() for simplicity; adjust as needed.
		if err != nil {
			if err == io.EOF {
				// The session was closed gracefully.
				fmt.Println("Session closed by client.")
				return
			}
			fmt.Printf("Error accepting stream: %v\n", err)
			return // Or continue to try accepting new streams, depending on your error handling strategy.
		}
		// For each file transfer, open a new confirmation stream.
		// This avoids sharing the same stream across multiple goroutines,
		// reducing complexity around synchronization and error handling.
		go func(s quic.Stream) {
			revieveFile(s)
		}(stream)
	}
}

func revieveFile(stream quic.Stream) {
	var decoder *BinaryCoder

	recieved := 0
	buffer := make([]byte, FRAMESIZE)

	// create a new buffer to store the incoming packets
	for {
		accu_recv := 0

		for accu_recv < FRAMESIZE {
			n, err := stream.Read(buffer[accu_recv:])

			if err != nil {
				if err == io.EOF {
					// EOF indicates the client closed the stream. All data has been received.
					fmt.Println("Stream closed by client")
					break
				}
				// Handle other errors that might occur during reading.
				fmt.Println("Error reading from stream:", err)
				continue // or handle the error appropriately
			} else {
				accu_recv += n
			}
		}

		xnc, err := DecodeXNCPkt(buffer)
		if err != nil {
			fmt.Println("Error decoding XNC packet:", err)
			continue
		}

		if decoder == nil {
			fmt.Printf("Start Recieve chunk %d\n", xnc.ChunkId)
			// Assuming InitBinaryCoder, PKTBITNUM, and RNGSEED are correctly defined elsewhere.
			decoder = InitBinaryCoder(SYMBOLNUM, PKTBITNUM, RNGSEED)
		}

		coefficient := UnpackUint64sToBinaryBytes(xnc.Coefficient, SYMBOLNUM)
		pkt := UnpackUint64sToBinaryBytes(xnc.Packet, PKTBITNUM)

		decoder.ConsumePacket(coefficient, pkt)

		recieved++

		if decoder.IsFullyDecoded() {
			fmt.Printf("## Received packets %v, Decode %d out of %d\n", recieved, decoder.GetNumDecoded(), decoder.NumSymbols)

			fmt.Print("## File fully received\n")

			file := PacketsToBytes(decoder.PacketVector, PKTBITNUM, CHUNKSIZE*8)
			file = file[:xnc.PktSize]

			filename := fmt.Sprintf("recv_%d.m4s", xnc.ChunkId)
			if err := os.WriteFile(filename, file, 0644); err != nil {
				fmt.Errorf("Failed to save file: %v\n", err)
				break
			}

			break
		}
	}

	fmt.Printf("## Finished receiving file\n")
}
