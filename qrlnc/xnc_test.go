package qrlnc

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

func TestWhole(t *testing.T) {
	var decoder *BinaryCoder
	var encoder *BinaryCoder

	file, err := os.Open("test.m4s")
	if err != nil {
		t.Errorf("Error opening file: %v", err)
		return
	}
	defer file.Close() // Ensure the file is closed after reading

	chunk := 1 << 20 // 1MB

	filebytes, err := io.ReadAll(file)
	if err != nil {
		t.Errorf("Error reading file: %v", err)
		return
	}
	fmt.Printf("Read %d bytes from file\n", len(filebytes))

	fmt.Printf("Chunk num %d\n", len(filebytes)/chunk)

	// for i := 0; i < len(bytes); i += chunk {
	end := min(0+chunk, len(filebytes))
	chunkBytes := filebytes[0:end]

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

	recieved := 0

	for {
		coefficientE, packet := encoder.GetNewCodedPacket()
		coefEu64, origLenCoef := BytesToUint64s(coefficientE)
		pktEu64, origLenPkt := BytesToUint64s(packet)

		xncE := XNC{
			ChunkId:     0,
			FileSize:    len(chunkBytes),
			NumSymbols:  encoder.NumSymbols,
			Coefficient: coefEu64,
			Packet:      pktEu64,
		}

		pktE, err := EncodeXNCToByte(xncE)
		if err != nil {
			t.Errorf("Error encoding packet data: %v", err)
			return
		}

		xncD, err := DecodeByteToXNC(pktE)

		if !XNCEqual(xncE, xncD) {
			t.Errorf("Failed to decode xnc correctly.\nExpected: %v\nGot: %v", xncE, xncD)
			return
		}

		if err != nil {
			t.Errorf("Error decoding packet data: %v", err)
			return
			// Decide on error handling strategy, possibly continue to the next stream.
		}

		if decoder == nil {
			// Assuming InitBinaryCoder, PktNumBit, and RngSeed are correctly defined elsewhere.
			decoder = InitBinaryCoder(xncD.NumSymbols, PktNumBit, 1)
		}

		coefficientD := Uint64sToBytes(xncD.Coefficient, origLenCoef)
		pktD := Uint64sToBytes(xncD.Packet, origLenPkt)

		if !bytes.Equal(coefficientE, coefficientD) {
			t.Errorf("Failed to decode coefficients correctly.\nExpected: %x\nGot: %x", coefficientE, coefficientD)
			return
		}

		decoder.ConsumePacket(coefficientD, pktD)

		recieved++

		t.Logf("## Received packets %v, Decode %d out of %d\n", recieved, decoder.GetNumDecoded(), decoder.NumSymbols)

		if decoder.IsFullyDecoded() {
			t.Logf("\n# Finished Decode!!!")

			if equal(decoder.PacketVector, encoder.PacketVector) {
				t.Logf("## Successfully decoded all packets at the receiver after %d messages.", recieved)
			} else {
				t.Error("## Error, decoded packet vectors are not equal!!!")
				return
			}

			file := PacketsToBytes(decoder.PacketVector, PktNumBit, len(chunkBytes)*8)

			if !bytes.Equal(file, chunkBytes) {
				t.Errorf("## Files and chunkBytes do not match.")
				return
			}

			if err := os.WriteFile("recv.m4s", file, 0644); err != nil {
				t.Errorf("Failed to save file: %v\n", err)
				return
			}

			break
		}
	}

	original, err := ioutil.ReadFile("test.m4s")
	if err != nil {
		t.Errorf("Error opening original file: %v", err)
		return
	}

	received, err := ioutil.ReadFile("recv.m4s")
	if err != nil {
		t.Errorf("Error opening received file: %v", err)
		return
	}

	t.Logf("## Original file size: %d bytes", len(original))
	t.Logf("## Received file size: %d bytes", len(received))

	if !bytes.Equal(original, received) {
		t.Errorf("## Files do not match.")
		return
	} else {
		t.Logf("## Successfully decoded all packets at the receiver.")
	}
}

func TestConversion(t *testing.T) {
	// Initialize a test case with a slice of bytes.
	// Ensure the length is a multiple of 8 for straightforward testing.
	originalBytes := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D}

	// Convert the bytes to uint64s
	uint64s, origLen := BytesToUint64s(originalBytes)

	// Convert back to bytes
	convertedBytes := Uint64sToBytes(uint64s, origLen)

	// Compare the original byte slice with the converted byte slice
	if !bytes.Equal(originalBytes, convertedBytes) {
		t.Errorf("Conversion failed. Original: %x, Converted: %x", originalBytes, convertedBytes)
	}
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

	if err := os.WriteFile("recv.m4s", decode, 0644); err != nil {
		t.Errorf("Failed to save file: %v\n", err)
	}
	original, err := ioutil.ReadFile("test.m4s")
	if err != nil {
		t.Logf("Error opening original file: %v", err)
	}

	received, err := ioutil.ReadFile("recv.m4s")
	if err != nil {
		t.Logf("Error opening received file: %v", err)
	}

	t.Logf("## Original file size: %d bytes", len(original))
	t.Logf("## Received file size: %d bytes", len(received))

	if !bytes.Equal(original, received) {
		t.Errorf("## Files do not match.")
	} else {
		t.Logf("## Successfully decoded all packets at the receiver.")
	}
}

func TestXNCToByte(t *testing.T) {
	xnc := XNC{
		ChunkId:     1,
		Type:        byte(3),
		NumSymbols:  10,
		FileSize:    4,
		Coefficient: []uint64{1, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		Packet:      []uint64{135431, 51908357, 1324951, 1587324, 1587324, 1587324, 1587324, 1587324, 1587324, 1587324},
	}

	encode, err := EncodeXNCToByte(xnc)
	if err != nil {
		fmt.Println("Error encode xnc:", err)
		return
	}
	decode, err := DecodeByteToXNC(encode)
	if err != nil {
		fmt.Println("Error decode xnc:", err)
		return
	}

	if !XNCEqual(xnc, decode) {
		t.Errorf("Failed to decode xnc correctly.\nExpected: %v\nGot: %v", xnc, decode)
	}

}

func XNCEqual(a, b XNC) bool {
	if a.ChunkId != b.ChunkId {
		return false
	}
	if a.Type != b.Type {
		return false
	}
	if a.NumSymbols != b.NumSymbols {
		return false
	}
	if a.FileSize != b.FileSize {
		return false
	}
	if !bytes.Equal(Uint64sToBytes(a.Coefficient, len(a.Coefficient)), Uint64sToBytes(b.Coefficient, len(b.Coefficient))) {
		return false
	}

	for i := range a.Packet {
		if a.Packet[i] != b.Packet[i] {
			return false
		}
	}

	return true
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
