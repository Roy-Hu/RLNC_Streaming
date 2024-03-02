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

var encoder *BinaryCoder
var decoder *BinaryCoder

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func TestPacketToByte(t *testing.T) {
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

	fmt.Printf("Read %d bytes from file\n", len(byts))
	encode := bytesToPackets(byts, pktNumBit)
	decode := packetsToBytes(encode, pktNumBit, len(byts)*8)

	if bytes.Equal(decode, byts) {
		t.Logf("## Successfully decoded all packets at the receiver after messages.")
	} else {
		t.Errorf("Failed to decode all packets correctly.\nExpected: %x\nGot: %x", byts, decode)
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

func generateTLSConfig() *tls.Config {
	cert, err := tls.LoadX509KeyPair("../godash/http/certs/cert.pem", "../godash/http/certs/key.pem")
	if err != nil {
		fmt.Printf("TLS config err: %v", err)

		return nil
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
}

func client(filename string) {
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
	packets := bytesToPackets(bytes, pktNumBit)

	numSymbols := len(packets)

	encoder = InitBinaryCoder(numSymbols, pktNumBit, rngSeed)

	fmt.Println("Number of symbols:", encoder.numSymbols)
	fmt.Println("Number of bit per packet:", encoder.numBitPacket)

	// Initialize encoder with random bit packets
	for packetID := 0; packetID < encoder.numSymbols; packetID++ {
		coefficients := make([]int, encoder.numSymbols)
		coefficients[packetID] = 1
		encoder.consumePacket(coefficients, packets[packetID])
	}

	for {
		coefficient, packet := encoder.getNewCodedPacket()

		xncPkt := XNC{
			BitSize:     len(bytes) * 8,
			NumSymbols:  encoder.numSymbols,
			Coefficient: coefficient,
			Packet:      packet,
		}

		encodedPkt, err := encodePacketDataToByte(xncPkt)

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
	tlsConf := generateTLSConfig()
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

		data, err = decodePacketDataToByte(buffer[:n])
		if err != nil {
			fmt.Println("Error decoding packet data:", err)
			// Decide on error handling strategy, possibly continue to the next stream.
		}

		if first {
			// Assuming InitBinaryCoder, pktNumBit, and rngSeed are correctly defined elsewhere.
			decoder = InitBinaryCoder(data.NumSymbols, pktNumBit, 1)
			first = false
		}

		decoder.consumePacket(data.Coefficient, data.Packet)

		fmt.Printf("Received packets %v\n", recieved)

		recieved++
		// Send acknowledgment back to the client.

		fmt.Printf("## Decode %d out of %d\n", decoder.getNumDecoded(), decoder.numSymbols)

		if decoder.isFullyDecoded() {

			img := packetsToBytes(decoder.packetVector, decoder.numBitPacket, data.BitSize)
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
