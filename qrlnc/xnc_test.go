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

	t.Log("## TestWhole")
	t.Logf("SYMBOLNUM %d", SYMBOLNUM)
	t.Logf("COEFNUM %d", COEFNUM)

	file, err := os.Open("test.m4s")
	if err != nil {
		t.Errorf("Error opening file: %v", err)
		return
	}
	defer file.Close() // Ensure the file is closed after reading

	filebytes, err := io.ReadAll(file)
	if err != nil {
		t.Errorf("Error reading file: %v", err)
		return
	}
	fmt.Printf("Read %d bytes from file\n", len(filebytes))

	fmt.Printf("Chunk num %d\n", len(filebytes)/CHUNKSIZE)

	// for i := 0; i < len(bytes); i += CHUNKSIZE {
	filesize := len(filebytes)

	end := min(0+CHUNKSIZE, len(filebytes))
	chunkBytes := filebytes[0:end]
	// // padding chunkbytes to chunk size
	if len(chunkBytes) < CHUNKSIZE {
		chunkBytes = append(chunkBytes, make([]byte, CHUNKSIZE-len(chunkBytes))...)
	}

	packets := BytesToPackets(chunkBytes, PKTBITNUM)

	encoder = InitBinaryCoder(len(packets), PKTBITNUM, RNGSEED)

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
		coefEu64, origLenCoef := PackBinaryBytesToUint64s(coefficientE)
		pktEu64, origLenPkt := PackBinaryBytesToUint64s(packet)

		if (len(coefEu64) != COEFNUM) || (origLenCoef != SYMBOLNUM) || (origLenPkt != PKTBITNUM) {
			t.Errorf("Error encoding packet data: invalid length")
			return
		}
		t.Logf("Pkt bit num %d, pky u64 len %d", PKTBITNUM, len(pktEu64))
		xncE := XNC{
			ChunkId:     0,
			FileSize:    filesize,
			Coefficient: coefEu64,
			Packet:      pktEu64,
		}

		pktE, err := EncodeXNCPkt(xncE)
		if err != nil {
			t.Errorf("Error encoding packet data: %v", err)
			return
		}

		xncD, err := DecodeXNCPkt(pktE)

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
			decoder = InitBinaryCoder(SYMBOLNUM, PKTBITNUM, 1)
		}

		coefficientD := UnpackUint64sToBinaryBytes(xncD.Coefficient, SYMBOLNUM)
		pktD := UnpackUint64sToBinaryBytes(xncD.Packet, PKTBITNUM)

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

			recvfile := PacketsToBytes(decoder.PacketVector, decoder.NumBitPacket, len(chunkBytes)*8)
			recvfile = recvfile[:xncD.FileSize]

			if !bytes.Equal(recvfile, filebytes) {
				t.Errorf("## recvfile and filebytes do not match.")
				return
			}

			if err := os.WriteFile("recv.m4s", recvfile, 0644); err != nil {
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

func TestEncodeXNCPkt(t *testing.T) {
	xnc := XNC{
		ChunkId:     1,
		Type:        byte(TYPE_XNC),
		FileSize:    4,
		Coefficient: []uint64{1342493851, 1238124},
		Packet:      make([]uint64, PKTNUM),
	}

	for i := range xnc.Packet {
		xnc.Packet[i] = uint64(i)
	}

	encode, err := EncodeXNCPkt(xnc)
	if err != nil {
		t.Errorf("Failed to encode xnc correctly.")
		return
	}

	decode, err := DecodeXNCPkt(encode)
	if err != nil {
		t.Errorf("Failed to encode xnc correctly.")
		return
	}

	if !XNCEqual(xnc, decode) {
		t.Errorf("Failed to decode xnc correctly.\nExpected: %v\nGot: %v", xnc, decode)
		return
	}

	t.Logf("## Successfully decoded XNC packets\n")
}

func TesTBinaryBtyeToUint64(t *testing.T) {
	// Initialize a test case with a slice of bytes.
	// Ensure the length is a multiple of 8 for straightforward testing.
	const length = 210 // Specify the desired length here.
	originalBinaryBytes := make([]byte, length)

	// Fill the slice with a simple pattern of 0x01 and 0x00 alternately.
	for i := 0; i < length; i++ {
		if i%2 == 0 || i%5 == 0 {
			originalBinaryBytes[i] = 0x01
		} else {
			originalBinaryBytes[i] = 0x00
		}
	}

	// Convert the bytes to uint64s
	uint64s, origLen := PackBinaryBytesToUint64s(originalBinaryBytes)

	// Convert back to bytes
	convertedBytes := UnpackUint64sToBinaryBytes(uint64s, origLen)

	// Compare the original byte slice with the converted byte slice
	if !bytes.Equal(originalBinaryBytes, convertedBytes) {
		t.Errorf("Conversion failed. Original: %x, Converted: %x", originalBinaryBytes, convertedBytes)
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
	encode := BytesToPackets(byts, PKTBITNUM)
	decode := PacketsToBytes(encode, PKTBITNUM, len(byts)*8)

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
		FileSize:    4,
		Coefficient: []uint64{1, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		Packet:      []uint64{135431, 51908357, 1324951, 1587324, 1587324, 1587324, 1587324, 1587324, 1587324, 1587324},
	}

	encode, err := EncodeXNCPkt(xnc)
	if err != nil {
		fmt.Println("Error encode xnc:", err)
		return
	}
	decode, err := DecodeXNCPkt(encode)
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
	if a.FileSize != b.FileSize {
		return false
	}
	if !bytes.Equal(UnpackUint64sToBinaryBytes(a.Coefficient, len(a.Coefficient)), UnpackUint64sToBinaryBytes(b.Coefficient, len(b.Coefficient))) {
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
