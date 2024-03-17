// package main

// import (
// 	"bytes"
// 	"context"
// 	"crypto/tls"
// 	"fmt"
// 	"io"
// 	"io/ioutil"
// 	"os"
// 	"time"

// 	"github.com/comp529/qrlnc"
// 	"github.com/lucas-clemente/quic-go"
// )

// func main() {
// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel() // Ensures the server goroutine is terminated.

// 	go server(ctx)
// 	time.Sleep(1 * time.Second) // Wait for the server to initialize.

// 	client("test.m4s")
// 	fmt.Printf("## Finished sending file")

// 	// wait for the server to finish
// 	time.Sleep(5 * time.Second)

// 	original, err := ioutil.ReadFile("test.m4s")
// 	if err != nil {
// 		fmt.Errorf("Error opening original file: %v", err)
// 	}

// 	received, err := ioutil.ReadFile("recv.m4s")
// 	if err != nil {
// 		fmt.Errorf("Error opening received file: %v", err)
// 	}

// 	fmt.Printf("## Original file size: %d bytes", len(original))
// 	fmt.Printf("## Received file size: %d bytes", len(received))

// 	if !bytes.Equal(original, received) {
// 		fmt.Errorf("## Files do not match.\nExpected: %x\nGot: %x", original, received)
// 	} else {
// 		fmt.Printf("## Successfully decoded all packets at the receiver.")
// 	}
// }

// func client(filename string) {
// 	var encoder *qrlnc.BinaryCoder

// 	quicConf := &quic.Config{}
// 	session, err := quic.DialAddr("localhost:4242", &tls.Config{InsecureSkipVerify: true}, quicConf)
// 	if err != nil {
// 		panic(err)
// 	}

// 	stream, err := session.OpenStreamSync()
// 	if err != nil {
// 		panic(err)
// 	}

// 	file, err := os.Open("test.m4s")
// 	if err != nil {
// 		fmt.Errorf("Error opening file: %v", err)
// 		return
// 	}

// 	defer file.Close() // Ensure the file is closed after reading

// 	chunk := 1 << 20 // 1MB

// 	filebytes, err := io.ReadAll(file)
// 	if err != nil {
// 		fmt.Errorf("Error reading file: %v", err)
// 		return
// 	}
// 	fmt.Printf("Read %d bytes from file\n", len(filebytes))

// 	fmt.Printf("Chunk num %d\n", len(filebytes)/chunk)

// 	// for i := 0; i < len(bytes); i += chunk {
// 	end := min(0+chunk, len(filebytes))
// 	chunkBytes := filebytes[0:end]

// 	packets := qrlnc.BytesToPackets(chunkBytes, qrlnc.PKTBITNUM)

// 	NumSymbols := len(packets)

// 	encoder = qrlnc.InitBinaryCoder(NumSymbols, qrlnc.PKTBITNUM, qrlnc.RNGSEED)

// 	fmt.Println("Number of symbols:", encoder.NumSymbols)
// 	fmt.Println("Number of bit per packet:", encoder.NumBitPacket)

// 	// Initialize encoder with random bit packets
// 	for packetID := 0; packetID < encoder.NumSymbols; packetID++ {
// 		coefficients := make([]byte, encoder.NumSymbols)
// 		coefficients[packetID] = 1
// 		encoder.ConsumePacket(coefficients, packets[packetID])
// 	}

// 	for {
// 		encodedPkt, err := encoder.GetNewCodedPacketByte(len(chunkBytes), 0)
// 		if err != nil {
// 			fmt.Println("Error encoding packet data:", err)
// 			return
// 		}

// 		_, err = stream.Write(encodedPkt)
// 		if err != nil {
// 			panic(err)
// 		}
// 		// }
// 	}
// }

// func min(a, b int) int {
// 	if a < b {
// 		return a
// 	}
// 	return b
// }
