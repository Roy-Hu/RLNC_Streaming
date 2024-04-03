package xnc

import (
	"bytes"
	"crypto/sha512"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"testing"

	"github.com/itzmeanjan/kodr"
	"github.com/itzmeanjan/kodr/full"
)

func TestWhole(t *testing.T) {
	t.Log("## TestWhole")

	file, err := os.Open("/var/www/html/tos_4sec_full/4K_dataset/4_sec/x264/bbb/DASH_Files/full/test.m4s")
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

	rFile := make([]byte, 0)

	chunks := SpiltFile(filebytes, CHUNKSIZE)

	var size int
	for i := 0; i < len(chunks); i++ {

		if i == len(chunks)-1 {
			size = len(filebytes) % CHUNKSIZE
		} else {
			size = CHUNKSIZE
		}

		hasher := sha512.New512_224()
		hasher.Write(chunks[i])

		enc, err := full.NewFullRLNCEncoderWithPieceCount(chunks[i], PIECECNT)
		if err != nil {
			log.Printf("Error: %s\n", err.Error())
			return
		}

		codedPieces := make([]*kodr.CodedPiece, 0, CODEDPIECECNT)
		for i := 0; i < int(CODEDPIECECNT); i++ {
			codedPieces = append(codedPieces, enc.CodedPiece())
		}

		decoder := full.NewFullRLNCDecoder(PIECECNT)

		for s := 0; s < int(CODEDPIECECNT); s++ {
			pktE, err := GetXNCEncPkt(size, i, len(chunks), codedPieces[s])
			if err != nil {
				t.Errorf("Error encoding packet data: %v", err)
				return
			}

			xncD, err := DecodeXNCPkt(pktE)
			if err != nil {
				t.Errorf("Error decoding packet data: %v", err)
				return
			}

			pieceD := &kodr.CodedPiece{
				Vector: xncD.Vector,
				Piece:  xncD.Piece,
			}

			if err := decoder.AddPiece(pieceD); err != nil {
				if errors.Is(err, kodr.ErrAllUsefulPiecesReceived) {
					fmt.Printf("All useful pieces received\n")
					break
				} else {
					t.Errorf("Error adding pieces: %v", err)
					return
				}
			}

			if decoder.IsDecoded() {
				t.Logf("\n# Finished Decode for chunk %d!!!", i)
				recvfile, err := GetFile(decoder)
				if err != nil {
					t.Errorf("Error geting file: %v", err)
					return
				}

				rFile = append(rFile, recvfile[:size]...)
			}
		}
	}

	t.Logf("## Original file size: %d bytes", len(filebytes))
	t.Logf("## Received file size: %d bytes", len(rFile))

	if !bytes.Equal(filebytes, rFile) {
		t.Errorf("## Files do not match.")
		return
	} else {
		t.Logf("## Successfully decoded all packets at the receiver.")
	}
}

func TestOriginXNCPkt(t *testing.T) {
	xnc := XNC{
		ChunkId:   1,
		Type:      byte(TYPE_XNC),
		ChunkSize: 4,
		ChunkNum:  10,
		Piece:     make([]byte, PIECESIZE),
	}

	for i := range xnc.Piece {
		xnc.Piece[i] = byte(i)
	}

	encode, err := EncodeXNCPkt(xnc)
	if err != nil {
		t.Errorf("Failed to encode xnc correctly: %v", err)
		return
	}

	decode, err := DecodeXNCPkt(encode)
	if err != nil {
		t.Errorf("Failed to encode xnc correctly: %v", err)
		return
	}

	if !XNCEqual(xnc, decode) {
		t.Errorf("Failed to decode xnc correctly.\nExpected: %v\nGot: %v", xnc, decode)
		return
	}

	t.Logf("## Successfully decoded Origin XNC packets\n")
}

func TestInit(t *testing.T) {
	init := XNC_INIT{
		Type:     TYPE_INIT_ENC,
		Len:      4,
		Filename: "test",
	}

	encode, err := EncodeInit(init)
	if err != nil {
		t.Errorf("Error encode xnc: %v", err)
		return
	}
	decode, err := DecodeInit(encode)
	if err != nil {
		t.Errorf("Error decode xnc: %v", err)
		return
	}

	if init.Type != decode.Type {
		t.Errorf("Failed to decode xnc correctly.\nExpected: %v\nGot: %v", init.Type, decode.Type)
	}
	if init.Len != decode.Len {
		t.Errorf("Failed to decode xnc correctly.\nExpected: %v\nGot: %v", init.Len, decode.Len)
	}
	if init.Filename != decode.Filename {
		t.Errorf("Failed to decode xnc correctly.\nExpected: %v\nGot: %v", init.Filename, decode.Filename)
	}
}
func TestXNC(t *testing.T) {
	xnc := XNC{
		ChunkId:   1,
		Type:      TYPE_XNC_ENC,
		ChunkSize: 4,
		ChunkNum:  10,
		Vector:    make([]byte, VECTORSIZE),
		Piece:     make([]byte, PIECESIZE),
	}

	for i := range xnc.Vector {
		xnc.Vector[i] = byte(i)
	}
	for i := range xnc.Piece {
		xnc.Piece[i] = byte(i)
	}

	encode, err := EncodeXNCPkt(xnc)
	if err != nil {
		t.Errorf("Error encode xnc: %v", err)
		return
	}
	decode, err := DecodeXNCPkt(encode)
	if err != nil {
		t.Errorf("Error decode xnc: %v", err)
		return
	}

	if !XNCEqual(xnc, decode) {
		t.Errorf("Failed to decode xnc correctly.\nExpected: %v\nGot: %v", xnc, decode)
		return
	}

	t.Logf("## Successfully decoded XNC packets\n")
}

func XNCEqual(a, b XNC) bool {
	if a.ChunkId != b.ChunkId {
		return false
	}
	if a.Type != b.Type {
		return false
	}
	if a.ChunkSize != b.ChunkSize {
		return false
	}

	if a.ChunkNum != b.ChunkNum {
		return false
	}
	if !bytes.Equal(a.Piece, b.Piece) || !bytes.Equal(a.Vector, b.Vector) {
		return false
	}

	return true
}
