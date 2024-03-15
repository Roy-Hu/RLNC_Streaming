package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/comp529/qrlnc"
	"github.com/lucas-clemente/quic-go"
)

func TestXNCEncodingDecoding(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure cancellation at the end of the test

	// Start the server in a goroutine
	go server(ctx)

	// Allow some time for the server to initialize
	time.Sleep(1 * time.Second)

	// Run the client to send the image
	client("pic.jpg")

	// compare pic.jpg and pic_recv.jpg
	file, err := os.Open("pic.jpg")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close() // Ensure the file is closed after reading

	byts, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	fileRecv, err := os.Open("pic_recv.jpg")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer fileRecv.Close() // Ensure the file is closed after reading

	bytsRecv, err := io.ReadAll(fileRecv)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	t.Logf("## Original file size: %d bytes", len(byts))
	t.Logf("## Received file size: %d bytes", len(bytsRecv))

	if bytes.Equal(byts, bytsRecv) {
		t.Logf("## Successfully decoded all packets at the receiver after messages.")
	} else {
		t.Errorf("## Failed to decode all packets at the receiver after messages.")
	}
}

func client(filename string) {
	var encoder *qrlnc.BinaryCoder

	quicConf := &quic.Config{}
	session, err := quic.DialAddr("localhost:4242", &tls.Config{InsecureSkipVerify: true}, quicConf)
	if err != nil {
		panic(err)
	}
	stream, err := session.OpenStreamSync()
	if err != nil {
		panic(err)
	}

	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close() // Ensure the file is closed after reading

	bytes, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	fmt.Printf("Read %d bytes from file\n", len(bytes))
	packets := qrlnc.BytesToPackets(bytes, qrlnc.PktNumBit)

	NumSymbols := len(packets)

	encoder = qrlnc.InitBinaryCoder(NumSymbols, qrlnc.PktNumBit, qrlnc.RngSeed)

	fmt.Println("Number of symbols:", encoder.NumSymbols)
	fmt.Println("Number of bit per packet:", encoder.NumBitPacket)

	// Initialize encoder with random bit packets
	for packetID := 0; packetID < encoder.NumSymbols; packetID++ {
		coefficients := make([]int, encoder.NumSymbols)
		coefficients[packetID] = 1
		encoder.ConsumePacket(coefficients, packets[packetID])
	}

	for {
		coefficient, packet := encoder.GetNewCodedPacket()

		xncPkt := qrlnc.XNC{
			BitSize:     len(bytes) * 8,
			NumSymbols:  encoder.NumSymbols,
			Coefficient: coefficient,
			Packet:      packet,
		}

		encodedPkt, err := qrlnc.EncodePacketDataToByte(xncPkt)

		if err != nil {
			fmt.Println("Error encoding packet data:", err)
			return
		}

		_, err = stream.Write(encodedPkt)
		if err != nil {
			panic(err)
		}

		buf := make([]byte, 4)
		_, err = stream.Read(buf)
		if err != nil {
			if err == io.EOF {
				// Handle end of stream; this is expected when the other side closes the stream.
				fmt.Println("Stream closed by server")
				break // Exit reading loop gracefully
			} else {
				panic(err) // Panic or handle other errors differently
			}
		}

		fmt.Printf("Received %v from server\n", string(buf))

		if string(buf) == "END" {
			fmt.Printf("Received END from server\n")
		}
	}
}

func server(ctx context.Context) {

	quicConf := &quic.Config{}
	tlsConf := qrlnc.GenerateTLSConfig()
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
	var first bool = true
	recieved := 0
	var decoder *qrlnc.BinaryCoder

	stream, err := sess.AcceptStream()
	if err != nil {
		if err == io.EOF {
			// Gracefully handle session closure.
			fmt.Println("Session closed by client.")
			return
		}
		fmt.Printf("Error accepting stream: %v\n", err)
		return
	}
	fmt.Println("New stream accepted.")

	for {
		var data qrlnc.XNC

		buffer := make([]byte, 4096) // A 4KB buffer
		n, err := stream.Read(buffer)
		if err != nil {
			if err == io.EOF {
				// EOF indicates the client closed the stream. All data has been received.
				fmt.Println("Stream closed by client")
				break // Exit the loop.
			}
			// Handle other errors that might occur during reading.
			fmt.Println("Error reading from stream:", err)
			return // or handle the error appropriately
		}

		data, err = qrlnc.DecodePacketDataToByte(buffer[:n])
		if err != nil {
			fmt.Println("Error decoding packet data:", err)
			// Decide on error handling strategy, possibly continue to the next stream.
		}

		if first {
			// Assuming InitBinaryCoder, PktNumBit, and RngSeed are correctly defined elsewhere.
			decoder = qrlnc.InitBinaryCoder(data.NumSymbols, qrlnc.PktNumBit, 1)
			first = false
		}

		decoder.ConsumePacket(data.Coefficient, data.Packet)

		fmt.Printf("Received packets %v\n", recieved)

		recieved++
		// Send acknowledgment back to the client.

		fmt.Printf("## Decode %d out of %d\n", decoder.GetNumDecoded(), decoder.NumSymbols)

		if decoder.IsFullyDecoded() {

			img := qrlnc.PacketsToBytes(decoder.PacketVector, decoder.NumBitPacket, data.BitSize)
			filename := fmt.Sprintf("pic_recv.jpg")
			if err := os.WriteFile(filename, img, 0644); err != nil {
				fmt.Printf("Failed to save image: %v\n", err)
				// Handle the error, such as notifying the client or logging.
				continue // Or break, based on your error handling policy.
			}

			fmt.Printf("Image saved as %s\n", filename)

			if _, err := stream.Write([]byte("END")); err != nil {
				fmt.Printf("Error sending acknowledgment: %v\n", err)
				// Decide on error handling strategy, possibly continue to the next stream.
			}

			// Reset or prepare for next image.
			break
		}

		if _, err := stream.Write([]byte("ACK")); err != nil {
			fmt.Printf("Error sending acknowledgment: %v\n", err)
			// Decide on error handling strategy, possibly continue to the next stream.
		}
	}

	// Properly close the stream after the loop
	if err := stream.Close(); err != nil {
		fmt.Printf("Error closing stream: %v\n", err)
	}

	time.Sleep(1 * time.Second)
	// Close the session if no more streams will be accepted, depending on your application logic
	if err := sess.Close(nil); err != nil {
		fmt.Printf("Error closing session: %v\n", err)
	}
}
