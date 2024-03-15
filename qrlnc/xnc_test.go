package qrlnc

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"
)

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
	encode := BytesToPackets(byts, PktNumBit)
	decode := PacketsToBytes(encode, PktNumBit, len(byts)*8)

	if bytes.Equal(decode, byts) {
		t.Logf("## Successfully decoded all packets at the receiver after messages.")
	} else {
		t.Errorf("Failed to decode all packets correctly.\nExpected: %x\nGot: %x", byts, decode)
	}

}
