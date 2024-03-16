package qrlnc

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/lucas-clemente/quic-go"
)

func TestConversion(t *testing.T) {
	// Initialize a test case with a slice of bytes.
	// Ensure the length is a multiple of 8 for straightforward testing.
	originalBytes := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D}

	// Convert the bytes to uint64s
	uint64s := bytesToUint64s(originalBytes)

	// Convert back to bytes
	convertedBytes := uint64sToBytes(uint64s, len(originalBytes))

	// Compare the original byte slice with the converted byte slice
	if !bytes.Equal(originalBytes, convertedBytes) {
		t.Errorf("Conversion failed. Original: %x, Converted: %x", originalBytes, convertedBytes)
	}
}

func TestXNCEncodingDecoding(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure cancellation at the end of the test

	// Start the server in a goroutine
	go server(ctx)

	// Allow some time for the server to initialize
	time.Sleep(1 * time.Second)

	// Run the client to send the image
	client("test.m4s")

	// compare test.m4s and pic_recv.jpg
	file, err := os.Open("test.m4s")
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

	fileRecv, err := os.Open("recv.m4s")
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
		t.Errorf("## Expected: %x", byts)
		t.Errorf("## Got: %x", bytsRecv)
	}
}

func client(filename string) {
	var encoder *BinaryCoder

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
	packets := BytesToPackets(bytes, PktNumBit)

	NumSymbols := len(packets)

	encoder = InitBinaryCoder(NumSymbols, PktNumBit, RngSeed)

	fmt.Println("Number of symbols:", encoder.NumSymbols)
	fmt.Println("Number of bit per packet:", encoder.NumBitPacket)

	// Initialize encoder with random bit packets
	for packetID := 0; packetID < encoder.NumSymbols; packetID++ {
		coefficients := make([]byte, encoder.NumSymbols)
		coefficients[packetID] = 1
		encoder.ConsumePacket(coefficients, packets[packetID])
	}

	for {
		encodedPkt, err := encoder.GetNewCodedPacketByte(len(bytes))
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
	var first bool = true
	recieved := 0
	var decoder *BinaryCoder

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
		var data XNC

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

		data, err = DecodeByteToPacketData(buffer[:n])
		if err != nil {
			fmt.Println("Error decoding packet data:", err)
			// Decide on error handling strategy, possibly continue to the next stream.
		}

		if first {
			// Assuming InitBinaryCoder, PktNumBit, and RngSeed are correctly defined elsewhere.
			decoder = InitBinaryCoder(data.NumSymbols, PktNumBit, 1)
			first = false
		}

		coef := uint64sToBytes(data.Coefficient, data.NumSymbols)
		decoder.ConsumePacket(coef, data.Packet)

		fmt.Printf("Received packets %v\n", recieved)

		recieved++
		// Send acknowledgment back to the client.

		fmt.Printf("## Decode %d out of %d\n", decoder.GetNumDecoded(), decoder.NumSymbols)

		if decoder.IsFullyDecoded() {
			bitsize := data.FileSize * 8
			img := PacketsToBytes(decoder.PacketVector, decoder.NumBitPacket, bitsize)
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func TestPacketToByte(t *testing.T) {
	file, err := os.Open("test.m4s")
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

	fmt.Printf("Read %d bytes from file\n", len(byts))
	encode := BytesToPackets(byts, PktNumBit)
	decode := PacketsToBytes(encode, PktNumBit, len(byts)*8)

	if bytes.Equal(decode, byts) {
		t.Logf("## Successfully decoded all packets at the receiver after messages.")
	} else {
		t.Errorf("Failed to decode all packets correctly.\nExpected: %x\nGot: %x", byts, decode)
	}

}
