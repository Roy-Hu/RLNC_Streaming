package qrlnc

import (
	"crypto/tls"
	"fmt"
	"io"
	"os"
	"strings"

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
		panic(err)
	}

	chkId := 0
	for i := 0; i < len(filebytes); i += CHUNKSIZE {
		stream, err := sess.OpenStreamSync()
		if err != nil {
			panic(err)
		}

		fmt.Printf("Open Stream for chunk %d\n", chkId)

		var encoder *BinaryCoder

		end := Min(i+CHUNKSIZE, i+len(filebytes[i:]))
		chunkBytes := filebytes[i:end]

		size := end - i

		// // padding chunkbytes to chunk size
		if len(chunkBytes) < CHUNKSIZE {
			chunkBytes = append(chunkBytes, make([]byte, CHUNKSIZE-len(chunkBytes))...)
		}
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
				Type:        TYPE_XNC,
				ChunkId:     chkId,
				PktSize:     size,
				Coefficient: coefu64,
				Packet:      pktu64,
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
					fmt.Printf("Stream closed by the server, stopping write operations.\n")
					break
				} else if strings.Contains(err.Error(), "closed stream") {
					fmt.Printf("Stream closed by the client, stopping write operations.\n")
					continue
				} else {
					// Handle other errors that might not necessitate stopping.
					fmt.Printf("Error writing to stream: %v\n", err)
					continue
				}
			}

			sent++
		}

		chkId++
	}

	fmt.Printf("## Finished sending file\n")
}
