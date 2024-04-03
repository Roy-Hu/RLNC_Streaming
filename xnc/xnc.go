package xnc

import (
	"encoding/binary"
	"fmt"

	"github.com/itzmeanjan/kodr"
	"github.com/itzmeanjan/kodr/full"
)

// 8192 bits = 1024 bytes
var RNGSEED int64 = int64(1)

var TYPE_ACK byte = 0x2
var TYPE_INIT_ENC byte = 0x3
var TYPE_INIT_ORG byte = 0x4
var TYPE_XNC_ENC byte = 0x5
var TYPE_XNC_ORG byte = 0x6

var TYPESIZE int = 1
var IDSIZE int = 4
var FILESIZESIZE int = 4

var VECTORSIZE int = CHUNKSIZE / 1024
var PIECESIZE int = CHUNKSIZE / VECTORSIZE

var FRAMESIZE_ENC int = TYPESIZE + IDSIZE + FILESIZESIZE + VECTORSIZE + PIECESIZE
var FRAMESIZE_ORG int = TYPESIZE + IDSIZE + FILESIZESIZE + PIECESIZE
var ACKSIZE int = TYPESIZE + IDSIZE
var INITSIZE int = 128
var INFOSIZE int = TYPESIZE + 4 + 4

var END_CHUNK int = 8192

var (
	PIECECNT      uint = uint(VECTORSIZE)
	CODEDPIECECNT uint = PIECECNT
)

func GetXNCPkt(size int, id int, codepiece *kodr.CodedPiece) ([]byte, error) {
	vec := make([]byte, 0)
	piece := make([]byte, 0)

	vec = append(vec, codepiece.Vector...)
	piece = append(piece, codepiece.Piece...)

	xncE := XNC{
		Type:      TYPE_XNC_ENC,
		ChunkId:   id,
		ChunkSize: size,
		Vector:    vec,
		Piece:     piece,
	}

	pktE, err := EncodeXNCPkt(xncE)
	if err != nil {
		return nil, fmt.Errorf("Error encoding packet data: %v", err)

	}

	return pktE, nil
}

func GetFile(decoder *full.FullRLNCDecoder) ([]byte, error) {
	dec_p, err := decoder.GetPieces()
	if err != nil {
		return nil, fmt.Errorf("Error getting pieces: %v", err)
	}

	decoded_data := make([]byte, 0)
	for i := 0; i < len(dec_p); i++ {
		decoded_data = append(decoded_data, dec_p[i]...)
	}

	recvfile := []byte(decoded_data)

	return recvfile, nil
}

type XNC struct {
	Type      byte
	ChunkId   int
	ChunkSize int
	Vector    []byte
	Piece     []byte
}

type XNC_ACK struct {
	Type    byte
	ChunkId int
}

type XNC_INIT struct {
	Type     byte
	Len      int
	Filename string
}

type XNC_INFO struct {
	Type     byte
	ChunkNum int
	FileSize int
}

func EncodeInfo(data XNC_INFO) ([]byte, error) {
	pkt := []byte{}

	pkt = append(pkt, data.Type)

	chunkNum := make([]byte, 4)
	binary.BigEndian.PutUint32(chunkNum, uint32(data.ChunkNum))
	pkt = append(pkt, chunkNum...)

	fileSize := make([]byte, 4)
	binary.BigEndian.PutUint32(fileSize, uint32(data.FileSize))
	pkt = append(pkt, fileSize...)

	return pkt, nil
}

func EncodeInit(data XNC_INIT) ([]byte, error) {
	pkt := make([]byte, 128)

	pkt[0] = data.Type
	binary.BigEndian.PutUint32(pkt[1:5], uint32(data.Len))

	for i := 0; i < data.Len; i++ {
		pkt[5+i] = data.Filename[i]
	}

	return pkt, nil
}

func DecodeInit(data []byte) (XNC_INIT, error) {
	if data[0] != TYPE_INIT_ENC && data[0] != TYPE_INIT_ORG {
		return XNC_INIT{}, fmt.Errorf("pkt type is not correct\n")
	}

	init := XNC_INIT{}
	init.Type = data[0]
	init.Len = int(binary.BigEndian.Uint32(data[1:5]))
	init.Filename = string(data[5 : 5+init.Len])

	return init, nil
}

// XNC pkt
// Type + ChunkId + FileSize+ Packet
func EncodeAck(data XNC_ACK) ([]byte, error) {
	pkt := []byte{}

	pkt = append(pkt, data.Type)

	chunkId := make([]byte, 4)
	binary.BigEndian.PutUint32(chunkId, uint32(data.ChunkId))
	pkt = append(pkt, chunkId...)

	return pkt, nil
}

func DecodeAck(data []byte) (XNC_ACK, error) {
	if len(data) != ACKSIZE {
		return XNC_ACK{}, fmt.Errorf("XNC pkt size is not correct\n")
	}

	if data[0] != TYPE_ACK {
		return XNC_ACK{}, fmt.Errorf("pkt type is not correct\n")
	}

	ack := XNC_ACK{}

	ack.Type = data[0]
	ack.ChunkId = int(binary.BigEndian.Uint32(data[1:5]))

	return ack, nil
}

func EncodeXNCPkt(data XNC) ([]byte, error) {
	if data.Type == TYPE_XNC_ENC && (len(data.Vector) != VECTORSIZE || len(data.Piece) != PIECESIZE) {
		return nil, fmt.Errorf("XNC Vector %d or Piece %d size is not correct\n", len(data.Vector), len(data.Piece))
	} else if data.Type == TYPE_XNC_ORG && len(data.Piece) != PIECESIZE {
		return nil, fmt.Errorf("XNC pkt byte size is not correct\n")
	}

	pkt := []byte{}

	pkt = append(pkt, data.Type)

	chunkId := make([]byte, 4)
	binary.BigEndian.PutUint32(chunkId, uint32(data.ChunkId))
	pkt = append(pkt, chunkId...)

	pktsize := make([]byte, 4)
	binary.BigEndian.PutUint32(pktsize, uint32(data.ChunkSize))
	pkt = append(pkt, pktsize...)

	if data.Type == TYPE_XNC_ENC {
		pkt = append(pkt, data.Vector[:VECTORSIZE]...)
		pkt = append(pkt, data.Piece[:PIECESIZE]...)
	} else if data.Type == TYPE_XNC_ORG {
		pkt = append(pkt, data.Piece[:PIECESIZE]...)
	}

	return pkt, nil
}

func DecodeXNCPkt(data []byte) (XNC, error) {
	if len(data) != FRAMESIZE_ENC && len(data) != FRAMESIZE_ORG {
		return XNC{}, fmt.Errorf("XNC pkt size %d is not correct\n", len(data))
	}

	xnc := XNC{}
	xnc.Type = data[0]
	xnc.ChunkId = int(binary.BigEndian.Uint32(data[1:5]))
	xnc.ChunkSize = int(binary.BigEndian.Uint32(data[5:9]))

	if xnc.Type == TYPE_XNC_ENC {
		xnc.Vector = data[9 : 9+VECTORSIZE]
		xnc.Piece = data[9+VECTORSIZE : 9+VECTORSIZE+PIECESIZE]
	} else if xnc.Type == TYPE_XNC_ORG {
		xnc.Piece = data[9:]
	} else {
		return XNC{}, fmt.Errorf("Unknow XNC type\n")
	}

	return xnc, nil
}
