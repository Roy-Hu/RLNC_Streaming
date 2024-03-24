package qrlnc

import (
	"crypto/tls"
	"fmt"
	"io"
	"os"

	"github.com/lucas-clemente/quic-go"
)

func Client(filename string) {
	file, err := os.Open("test.m4s")
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
	initpkt, err := EncodeInit(XNC_INIT{
		Type:     TYPE_INIT,
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

	decoder := make(map[int]*BinaryCoder)

	recieved := 0
	chunkNum := 0
	buffer := make([]byte, FRAMESIZE)
	recvfile := []byte{}

	for {
		accu_recv := 0

		for accu_recv < FRAMESIZE {
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
			}
		}

		xnc, err := DecodeXNCPkt(buffer)
		if err != nil {
			fmt.Println("Error decoding XNC packet:", err)
			continue
		}

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
		pkt := UnpackUint64sToBinaryBytes(xnc.Packet, PKTBITNUM)

		decoder[xnc.ChunkId].ConsumePacket(coefficient, pkt)

		recieved++

		if decoder[xnc.ChunkId].IsFullyDecoded() {
			fmt.Printf("## Received packets %v, Decode %d out of %d\n", recieved, decoder[xnc.ChunkId].GetNumDecoded(), decoder[xnc.ChunkId].NumSymbols)

			fmt.Print("## File fully received\n")

			file := PacketsToBytes(decoder[xnc.ChunkId].PacketVector, PKTBITNUM, CHUNKSIZE*8)
			file = file[:xnc.ChunkSize]

			recvfile = append(recvfile, file...)

			if xnc.ChunkId == END_CHUNK {
				fmt.Printf("## Last chunk recieved\n")
				if err := os.WriteFile("recv.m4s", recvfile, 0644); err != nil {
					fmt.Errorf("Failed to save file: %v\n", err)
					return
				}

				break
			}
		}
	}

	fmt.Printf("## Finished recieving file\n")
}
