// package qrlnc

// import (
// 	"context"
// 	"fmt"
// 	"io"
// 	"os"

// 	"github.com/lucas-clemente/quic-go"
// )

// func Server(ctx context.Context) {
// 	quicConf := &quic.Config{}
// 	tlsConf := GenerateTLSConfig()
// 	if tlsConf == nil {
// 		return
// 	}

// 	listener, err := quic.ListenAddr("localhost:4242", tlsConf, quicConf)
// 	if err != nil {
// 		fmt.Println("Failed to start server:", err)
// 		return
// 	}
// 	defer listener.Close()

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			fmt.Println("Server shutting down")
// 			return
// 		default:
// 			sess, err := listener.Accept()
// 			if err != nil {
// 				fmt.Println("Failed to accept session:", err)
// 				return
// 			}
// 			defer sess.Close(nil)

// 			go handleSession(sess)
// 		}
// 	}
// }

// func revieveFile(stream quic.Stream) {
// 	var decoder *BinaryCoder

// 	recieved := 0

// 	// create a new buffer to store the incoming packets
// 	for {
// 		buffer := make([]byte, 8192) // A 4KB buffer
// 		n, err := stream.Read(buffer)
// 		rawpkt := buffer[:n]

// 		if err != nil {
// 			if err == io.EOF {
// 				// EOF indicates the client closed the stream. All data has been received.
// 				fmt.Println("Stream closed by client")
// 				break
// 			}
// 			// Handle other errors that might occur during reading.
// 			fmt.Println("Error reading from stream:", err)
// 			continue // or handle the error appropriately
// 		}

// 		if _, err := stream.Write([]byte("ACK")); err != nil {
// 			fmt.Printf("Error sending acknowledgment: %v\n", err)
// 			// Decide on error handling strategy, possibly continue to the next stream.
// 		}

// 		xnc, err := DecodeByteToXNC(rawpkt)

// 		if decoder == nil {
// 			// Assuming InitBinaryCoder, PKTBITNUM, and RNGSEED are correctly defined elsewhere.
// 			decoder = InitBinaryCoder(SYMBOLNUM, PKTBITNUM, 1)
// 		}

// 		coefficient := UnpackUint64sToBinaryBytes(xnc.Coefficient, SYMBOLNUM)
// 		pkt := UnpackUint64sToBinaryBytes(xnc.Packet, PKTBITNUM)

// 		decoder.ConsumePacket(coefficient, pkt)

// 		recieved++

// 		if decoder.IsFullyDecoded() {
// 			file := PacketsToBytes(decoder.PacketVector, PKTBITNUM, decoder.FileSize*8)

// 			if err := os.WriteFile("recv.m4s", file, 0644); err != nil {
// 				fmt.Errorf("Failed to save file: %v\n", err)
// 				return
// 			}

// 			break
// 		}
// 	}

// 	fmt.Printf("## Finished receiving file\n")
// }

// func handleSession(sess quic.Session) {

// 	stream, err := sess.AcceptStream()
// 	if err != nil {
// 		if err == io.EOF {
// 			// Gracefully handle session closure.
// 			fmt.Println("Session closed by client.")
// 			return
// 		}
// 		fmt.Printf("Error accepting stream: %v\n", err)
// 		return
// 	}
// 	fmt.Println("New stream accepted.")

// 	revieveFile(stream)
// }
