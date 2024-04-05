package qrlnc

import (
	"crypto/tls"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/lucas-clemente/quic-go"
)

func Client(filename string, encode bool) ([]byte, time.Duration, float64) {

	quicConf := &quic.Config{}
	sess, err := quic.DialAddr("localhost:4242", &tls.Config{InsecureSkipVerify: true}, quicConf)
	if err != nil {
		fmt.Errorf("Error dialing server: %v", err)
	}

	stream, err := sess.OpenStreamSync()
	if err != nil {
		fmt.Errorf("Error opening stream: %v", err)
	}

	fmt.Printf("filename len: %d\n", len(filename))

	var initType byte
	var frameSize int

	if encode {
		initType = TYPE_INIT_ENC
		frameSize = FRAMESIZE_ENC
	} else {
		initType = TYPE_INIT_ORG
		frameSize = FRAMESIZE_ORG
	}

	initpkt, err := EncodeInit(XNC_INIT{
		Type:     initType,
		Len:      len(filename),
		Filename: filename,
	})
	if err != nil {
		fmt.Errorf("Error encoding init packet: %v", err)
	}

	// Send the filename
	_, err = stream.Write(initpkt)
	if err != nil {
		fmt.Errorf("Error writing init packet: %v", err)
	}

	fmt.Printf("Sent init packet\n")

	decoder := make(map[int]*BinaryCoder)
	files := make(map[int][]byte)

	recieved := 0
	chunkNum := 0
	buffer := make([]byte, frameSize)
	recvfile := []byte{}
	startTime := time.Now()
	totalBytesRead := 0

	for {
		accu_recv := 0

		for accu_recv < frameSize {
			n, err := stream.Read(buffer[accu_recv:])
			if err != nil {
				if err == io.EOF {
					fmt.Errorf("Stream closed by server")
					break
				}
				fmt.Println("Error reading from stream:", err)
				continue // or handle the error appropriately
			} else {
				accu_recv += n
				totalBytesRead += n
			}
		}

		xnc, err := DecodeXNCPkt(buffer)
		if err != nil {
			fmt.Println("Error decoding XNC packet:", err)
			continue
		}

		recieved++

		if encode {
			if decoder[xnc.ChunkId] == nil {
				fmt.Printf("Start Recieve chunk %d\n", xnc.ChunkId)
				// Assuming InitBinaryCoder, PKTBITNUM, and RNGSEED are correctly defined elsewhere.
				decoder[xnc.ChunkId] = InitBinaryCoder(SYMBOLNUM, PKTBITNUM, RNGSEED)
				chunkNum++
			} else if decoder[xnc.ChunkId].IsFullyDecoded() {
				fmt.Printf("Chunk %d is already fully decoded\n", xnc.ChunkId)
				continue
			}

			coefficient := UnpackUint64sToBinaryBytes(xnc.Coefficient, SYMBOLNUM)
			pkt := UnpackUint64sToBinaryBytes(xnc.PktU64, PKTBITNUM)

			decoder[xnc.ChunkId].ConsumePacket(coefficient, pkt)

			if decoder[xnc.ChunkId].IsFullyDecoded() {
				fmt.Printf("## Received packets %v, Decode %d out of %d\n", recieved, decoder[xnc.ChunkId].GetNumDecoded(), decoder[xnc.ChunkId].NumSymbols)

				fmt.Print("## File fully received\n")

				file := PacketsToBytes(decoder[xnc.ChunkId].PacketVector, PKTBITNUM, CHUNKSIZE*8)
				file = file[:xnc.ChunkSize]
				recvfile = append(recvfile, file...)

				if xnc.ChunkId == END_CHUNK {
					fmt.Printf("## Last chunk recieved\n")
					break
				}
			}
		} else {
			files[xnc.ChunkId] = append(files[xnc.ChunkId], xnc.PktByte...)

			if len(files[xnc.ChunkId]) >= xnc.ChunkSize {
				recvfile = append(recvfile, files[xnc.ChunkId][:xnc.ChunkSize]...)

				if xnc.ChunkId == END_CHUNK {
					fmt.Printf("## Last chunk recieved\n")
					break
				}
			}
		}
	}

	stream.Context().Done()

	if err := os.WriteFile(filename, recvfile, 0644); err != nil {
		fmt.Errorf("Failed to save file: %v\n", err)

		duration := time.Since(startTime).Seconds()
		kbps := float64(totalBytesRead*8) / duration / 1024
		fmt.Printf("Received data at %.2f kbps\n", kbps)

		return nil, sess.GetRtt(), kbps
	}

	duration := time.Since(startTime).Seconds()
	kbps := float64(totalBytesRead*8) / duration / 1024
	fmt.Printf("Received data at %.2f kbps\n", kbps)

	fmt.Printf("## Finished recieving file\n")
	fmt.Printf("## Rtt %v\n", sess.GetRtt())

	return recvfile, sess.GetRtt(), kbps
}