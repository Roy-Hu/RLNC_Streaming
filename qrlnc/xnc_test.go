package qrlnc

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
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
	defer cancel() // Ensures the server goroutine is terminated.

	go server(ctx)
	time.Sleep(1 * time.Second) // Wait for the server to initialize.

	client("test.m4s")
	t.Log("## Finished sending file")

	// wait for the server to finish
	time.Sleep(5 * time.Second)

	original, err := ioutil.ReadFile("test.m4s")
	if err != nil {
		t.Fatalf("Error opening original file: %v", err)
	}

	received, err := ioutil.ReadFile("recv.m4s")
	if err != nil {
		t.Fatalf("Error opening received file: %v", err)
	}

	t.Logf("## Original file size: %d bytes", len(original))
	t.Logf("## Received file size: %d bytes", len(received))

	if !bytes.Equal(original, received) {
		t.Errorf("## Files do not match.\nExpected: %x\nGot: %x", original, received)
	} else {
		t.Log("## Successfully decoded all packets at the receiver.")
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

	chunk := 1 << 15 // 1MB

	bytes, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}
	fmt.Printf("Read %d bytes from file\n", len(bytes))

	fmt.Printf("Chunk num %d\n", len(bytes)/chunk)

	// for i := 0; i < len(bytes); i += chunk {
	end := min(0+chunk, len(bytes))
	chunkBytes := bytes[0:end]

	packets := BytesToPackets(chunkBytes, PktNumBit)

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
		encodedPkt, err := encoder.GetNewCodedPacketByte(len(chunkBytes), 0)
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
				fmt.Errorf("Error reading from stream: %v", err)
				continue
			}
		}
		fmt.Printf("Received %v from server\n", string(buf))
		// }
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
			defer sess.Close(nil)

			go handleSession(sess)
		}
	}
}

func revieveFile(stream quic.Stream) {
	var first bool = true
	recieved := 0
	var decoder *BinaryCoder

	// create a new buffer to store the incoming packets
	for {
		buffer := make([]byte, 8192) // A 4KB buffer
		n, err := stream.Read(buffer)
		if err != nil {
			if err == io.EOF {
				// EOF indicates the client closed the stream. All data has been received.
				fmt.Println("Stream closed by client")
				break
			}
			// Handle other errors that might occur during reading.
			fmt.Println("Error reading from stream:", err)
			continue // or handle the error appropriately
		}

		if _, err := stream.Write([]byte("ACK")); err != nil {
			fmt.Printf("Error sending acknowledgment: %v\n", err)
			// Decide on error handling strategy, possibly continue to the next stream.
		}

		data, err := DecodeByteToPacketData(buffer[:n])

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

		fmt.Printf("## Decode %d out of %d\n", decoder.GetNumDecoded(), decoder.NumSymbols)

		if decoder.IsFullyDecoded() {
			bitsize := data.FileSize * 8
			file := PacketsToBytes(decoder.PacketVector, decoder.NumBitPacket, bitsize)
			filename := fmt.Sprintf("recv.m4s")
			if err := os.WriteFile(filename, file, 0644); err != nil {
				fmt.Printf("Failed to save file: %v\n", err)
				// Handle the error, such as notifying the client or logging.
				continue // Or break, based on your error handling policy.
			}

			if _, err := stream.Write([]byte("END")); err != nil {
				fmt.Printf("Error sending acknowledgment: %v\n", err)
				// Decide on error handling strategy, possibly continue to the next stream.
			}

			// Properly close the stream after the loop
			if err := stream.Close(); err != nil {
				fmt.Printf("Error closing stream: %v\n", err)
			}

			time.Sleep(1 * time.Second)

			break
		}
	}

	fmt.Printf("## Finished receiving file\n")
}

func handleSession(sess quic.Session) {

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

	revieveFile(stream)
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
