package qrlnc

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/lucas-clemente/quic-go"
)

func Server(ctx context.Context, rootDir string) {
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

			go handleSession(sess, rootDir)
		}
	}
}

func handleSession(sess quic.Session, rootDir string) {
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

		fmt.Println("Stream accepted, waiting for init packet...")

		accu_recv := 0
		buffer := make([]byte, INITSIZE)
		for accu_recv < INITSIZE {
			n, err := stream.Read(buffer[accu_recv:])
			fmt.Printf("Read %d bytes\n", n)
			if err != nil {
				if err == io.EOF {
					fmt.Errorf("Stream closed by server")
					break
				}
				fmt.Println("Error reading from stream:", err)
				continue // or handle the error appropriately
			} else {
				accu_recv += n
			}
		}

		init, err := DecodeInit(buffer)
		if err != nil {
			fmt.Errorf("Error decoding init packet: %v", err)
			return
		}
		fmt.Printf("Init Filename: %v\n", init.Filename)
		filepath := filepath.Join(rootDir, init.Filename)

		if init.Type == TYPE_INIT_ENC {
			sendFile(stream, filepath, true)
		} else {
			sendFile(stream, filepath, false)
		}
	}
}

func sendFile(stream quic.Stream, filename string, encode bool) {
	fmt.Printf("Send file %s\n", filename)

	file, err := os.Open(filename)
	if err != nil {
		fmt.Errorf("Error opening file: %v", err)
		return
	}

	defer file.Close() // Ensure the file is closed after reading

	filebytes, err := io.ReadAll(file)
	if err != nil {
		fmt.Errorf("Error reading file: %v", err)
		return
	}
	fmt.Printf("Read %d bytes from file\n", len(filebytes))
	fmt.Printf("Chunk num %d\n", len(filebytes)/CHUNKSIZE)

	var encoder *BinaryCoder
	chkId := 0

	for i := 0; i < len(filebytes); i += CHUNKSIZE {
		fmt.Printf("Send Chunk %d\n", chkId)
		if i+CHUNKSIZE > len(filebytes) {
			fmt.Printf("Last chunk\n")
			chkId = END_CHUNK
		}

		end := Min(i+CHUNKSIZE, i+len(filebytes[i:]))
		chunkBytes := filebytes[i:end]

		size := end - i

		// // padding chunkbytes to chunk size
		if len(chunkBytes) < CHUNKSIZE {
			chunkBytes = append(chunkBytes, make([]byte, CHUNKSIZE-len(chunkBytes))...)
		}

		if encode {
			packets := BytesToPackets(chunkBytes, PKTBITNUM)
			encoder = InitBinaryCoder(len(packets), PKTBITNUM, RNGSEED)

			// Initialize encoder with random bit packets
			for packetID := 0; packetID < encoder.NumSymbols; packetID++ {
				coefficients := make([]byte, encoder.NumSymbols)
				coefficients[packetID] = 1
				encoder.ConsumePacket(coefficients, packets[packetID])
			}

			sent := 0

			for s := 0; s < len(packets); s++ {
				coefficient, packet := encoder.GetNewCodedPacket()
				coefu64, origLenCoef := PackBinaryBytesToUint64s(coefficient)
				pktu64, origLenPkt := PackBinaryBytesToUint64s(packet)

				if (len(coefu64) != COEFNUM) || (origLenCoef != SYMBOLNUM) || (origLenPkt != PKTBITNUM) {
					fmt.Errorf("Error encoding packet data: invalid length")
					continue
				}

				xnc := XNC{
					Type:        TYPE_XNC_ENC,
					ChunkId:     chkId,
					ChunkSize:   size,
					Coefficient: coefu64,
					PktU64:      pktu64,
				}

				pkt, err := EncodeXNCPkt(xnc)
				if err != nil {
					fmt.Errorf("Error encoding packet data: %v", err)
					continue
				}

				_, err = stream.Write(pkt)
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

				sent++
			}
		} else {
			for i := 0; i < CHUNKSIZE; i += PKTBITNUM / 8 {
				xnc := XNC{
					Type:      TYPE_XNC_ORG,
					ChunkId:   chkId,
					ChunkSize: size,
					PktByte:   chunkBytes[i : i+PKTBITNUM/8],
				}

				pkt, err := EncodeXNCPkt(xnc)
				if err != nil {
					fmt.Errorf("Error encoding packet data: %v", err)
					continue
				}

				_, err = stream.Write(pkt)
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

		chkId++
	}

	fmt.Printf("## Finished sending file\n")
}

func revieveFile(stream quic.Stream) {
	var decoder *BinaryCoder

	recieved := 0
	buffer := make([]byte, FRAMESIZE_ENC)

	// create a new buffer to store the incoming packets
	for {
		accu_recv := 0

		for accu_recv < FRAMESIZE_ENC {
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
		pkt := UnpackUint64sToBinaryBytes(xnc.PktU64, PKTBITNUM)

		decoder.ConsumePacket(coefficient, pkt)

		recieved++

		if decoder.IsFullyDecoded() {
			fmt.Printf("## Received packets %v, Decode %d out of %d\n", recieved, decoder.GetNumDecoded(), decoder.NumSymbols)

			fmt.Print("## File fully received\n")

			file := PacketsToBytes(decoder.PacketVector, PKTBITNUM, CHUNKSIZE*8)
			file = file[:xnc.ChunkSize]

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
