package xnc

import (
	"encoding/binary"
	"fmt"

	"github.com/itzmeanjan/kodr"
	"github.com/itzmeanjan/kodr/full"
)

// 8192 bits = 1024 bytes
var RNGSEED int64 = int64(1)
var TYPE_INIT_ENC byte = 0x3
var TYPE_INIT byte = 0x4
var TYPE_XNC_ENC byte = 0x5
var TYPE_XNC byte = 0x6
var TYPE_END byte = 0x7

var TYPESIZE int = 1
var IDSIZE int = 4
var NUMSIZE int = 4
var FILESIZESIZE int = 4

var VECTORSIZE int = CHUNKSIZE / 1024
var PIECESIZE int = CHUNKSIZE / VECTORSIZE

var FRAMESIZE_ENC int = TYPESIZE + IDSIZE + NUMSIZE + FILESIZESIZE + VECTORSIZE + PIECESIZE
var FRAMESIZE int = TYPESIZE + IDSIZE + NUMSIZE + FILESIZESIZE + PIECESIZE
var INITSIZE int = 128
var INFOSIZE int = TYPESIZE + 4 + 4

var (
	PIECECNT      uint = uint(VECTORSIZE)
	CODEDPIECECNT uint = PIECECNT + 1
	// loss debug
	// CODEDPIECECNT uint = PIECECNT*2
)

func GetXNCPkt(size int, id int, chunknum int, codepiece []byte) ([]byte, error) {
	xncE := XNC{
		Type:      TYPE_XNC,
		ChunkId:   id,
		ChunkSize: size,
		ChunkNum:  chunknum,
		Piece:     codepiece,
	}

	pktE, err := EncodeXNCPkt(xncE)
	if err != nil {
		return nil, fmt.Errorf("Error encoding packet data: %v", err)

	}

	return pktE, nil
}

func GetXNCEncPkt(size int, id int, chunknum int, codepiece *kodr.CodedPiece) ([]byte, error) {
	vec := make([]byte, 0)
	piece := make([]byte, 0)

	vec = append(vec, codepiece.Vector...)
	piece = append(piece, codepiece.Piece...)

	xncE := XNC{
		Type:      TYPE_XNC_ENC,
		ChunkId:   id,
		ChunkSize: size,
		ChunkNum:  chunknum,
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
	ChunkNum  int
	ChunkSize int
	Vector    []byte
	Piece     []byte
	End       bool
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
	if data[0] != TYPE_INIT_ENC && data[0] != TYPE_INIT {
		return XNC_INIT{}, fmt.Errorf("pkt type is not correct\n")
	}

	init := XNC_INIT{}
	init.Type = data[0]
	init.Len = int(binary.BigEndian.Uint32(data[1:5]))
	init.Filename = string(data[5 : 5+init.Len])

	return init, nil
}

func EncodeEND(id int, encode bool) []byte {
	var pkt []byte

	if encode {
		pkt = make([]byte, FRAMESIZE_ENC)
	} else {
		pkt = make([]byte, FRAMESIZE)
	}

	pkt[0] = TYPE_END
	binary.BigEndian.PutUint32(pkt[1:5], uint32(id))

	return pkt
}

func DecodeEND(pkt []byte, encode bool) (int, error) {
	if encode && len(pkt) != FRAMESIZE_ENC {
		return -1, fmt.Errorf("enc ack len is not correct\n")
	} else if !encode && len(pkt) != FRAMESIZE {
		return -1, fmt.Errorf("ack len is not correct\n")

	}

	if pkt[0] != TYPE_END {
		return -1, fmt.Errorf("pkt type is not correct\n")
	}

	id := int(binary.BigEndian.Uint32(pkt[1:5]))

	return id, nil
}

func EncodeXNCPkt(data XNC) ([]byte, error) {
	if data.Type == TYPE_XNC_ENC && (len(data.Vector) != VECTORSIZE || len(data.Piece) != PIECESIZE) {
		return nil, fmt.Errorf("XNC Vector %d or Piece %d size is not correct\n", len(data.Vector), len(data.Piece))
	} else if data.Type == TYPE_XNC && len(data.Piece) != PIECESIZE {
		return nil, fmt.Errorf("XNC pkt byte size is not correct\n")
	}

	pkt := make([]byte, 0, FRAMESIZE_ENC)

	pkt = append(pkt, data.Type)

	chunkId := make([]byte, 4)
	binary.BigEndian.PutUint32(chunkId, uint32(data.ChunkId))
	pkt = append(pkt, chunkId...)

	pktsize := make([]byte, 4)
	binary.BigEndian.PutUint32(pktsize, uint32(data.ChunkSize))
	pkt = append(pkt, pktsize...)

	chunknumsize := make([]byte, 4)
	binary.BigEndian.PutUint32(chunknumsize, uint32(data.ChunkNum))
	pkt = append(pkt, chunknumsize...)

	if data.Type == TYPE_XNC_ENC {
		pkt = append(pkt, data.Vector[:VECTORSIZE]...)
		pkt = append(pkt, data.Piece[:PIECESIZE]...)
	} else if data.Type == TYPE_XNC {
		pkt = append(pkt, data.Piece[:PIECESIZE]...)
	}

	return pkt, nil
}

func DecodeXNCPkt(data []byte) (XNC, error) {
	if len(data) != FRAMESIZE_ENC && len(data) != FRAMESIZE {
		return XNC{}, fmt.Errorf("XNC pkt size %d is not correct\n", len(data))
	}

	xnc := XNC{}
	xnc.Type = data[0]

	if xnc.Type == TYPE_END {
		return xnc, nil
	}

	xnc.ChunkId = int(binary.BigEndian.Uint32(data[1:5]))
	xnc.ChunkSize = int(binary.BigEndian.Uint32(data[5:9]))
	xnc.ChunkNum = int(binary.BigEndian.Uint32(data[9:13]))

	if xnc.Type == TYPE_XNC_ENC {
		xnc.Vector = data[13 : 13+VECTORSIZE]
		xnc.Piece = data[13+VECTORSIZE : 13+VECTORSIZE+PIECESIZE]
	} else if xnc.Type == TYPE_XNC {
		xnc.Piece = data[13:]
	} else {
		return XNC{}, fmt.Errorf("Unknow XNC type\n")
	}

	return xnc, nil
}
