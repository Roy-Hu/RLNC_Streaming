package xnc

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/itzmeanjan/kodr"
	"github.com/itzmeanjan/kodr/full"
	"github.com/lucas-clemente/quic-go"
)

func Client(filename string, encode bool) ([]byte, time.Duration, float64) {
	rand.Seed(42)

	fmt.Printf("[Client] Starting client, request file %v\n", filename)
	quicConf := &quic.Config{}
	sess, err := quic.DialAddr(serveraddr, &tls.Config{InsecureSkipVerify: true}, quicConf)
	if err != nil {
		fmt.Printf("[Client] Error dialing server: %v", err)
	}

	stream, err := sess.OpenStreamSync()
	if err != nil {
		fmt.Printf("[Client] Error opening stream: %v", err)
	}

	var initType byte
	var frameSize int

	initType = TYPE_INIT_ENC
	frameSize = FRAMESIZE_ENC

	initpkt, err := EncodeInit(XNC_INIT{
		Type:     initType,
		Len:      len(filename),
		Filename: filename,
	})
	if err != nil {
		fmt.Printf("[Client] Error encoding init packet: %v", err)
	}

	// Send the filename
	_, err = stream.Write(initpkt)
	if err != nil {
		fmt.Printf("[Client] Error writing init packet: %v", err)
	}

	startTime := time.Now()
	totalBytesRead := 0

	rFile := make([]byte, 0)

	decoders := make(map[int]*full.FullRLNCDecoder)

	fin := false

	for !fin {
		accu_recv := 0
		pktE := make([]byte, frameSize)

		for accu_recv < frameSize {
			n, err := stream.Read(pktE[accu_recv:])
			if err != nil {
				if err == io.EOF {
					fmt.Errorf("[Client] Stream closed by server")
					break
				}
				fmt.Println("[Client] Error reading from stream:", err)
				continue // or handle the error appropriately
			} else {
				accu_recv += n
				totalBytesRead += n
			}
		}

		xncD, err := DecodeXNCPkt(pktE)
		if err != nil {
			fmt.Printf("[Client] Error decoding packet data: %v", err)
			return nil, sess.GetRtt(), 0
		}

		pieceD := &kodr.CodedPiece{
			Vector: xncD.Vector,
			Piece:  xncD.Piece,
		}

		if _, ok := decoders[xncD.ChunkId]; !ok {
			decoders[xncD.ChunkId] = full.NewFullRLNCDecoder(PIECECNT)
		}

		if err := decoders[xncD.ChunkId].AddPiece(pieceD); err != nil {
			if errors.Is(err, kodr.ErrAllUsefulPiecesReceived) {
				fmt.Printf("[Client] All useful pieces received\n")
				break
			} else {
				fmt.Printf("[Client] Error adding pieces: %v", err)
				return nil, sess.GetRtt(), 0
			}
		}

		if decoders[xncD.ChunkId].IsDecoded() {
			recvfile, err := GetFile(decoders[xncD.ChunkId])
			if err != nil {
				fmt.Printf("[Client] Error geting file: %v", err)
				return nil, sess.GetRtt(), 0
			}

			rFile = append(rFile, recvfile[:xncD.ChunkSize]...)

			if xncD.ChunkId == END_CHUNK {
				fin = true
			}
		}
	}

	stream.Context().Done()
	stream.Close()

	duration := float64(time.Since(startTime).Microseconds()) / 1000000.0
	kbps := float64(totalBytesRead*8) / duration / 1000.
	fmt.Printf("[Client] Received data at %.2f kbps\n", kbps)
	fmt.Printf("[Client] Rtt %v\n", sess.GetRtt())
	fmt.Printf("[Client] Finished recieving file\n")

	return rFile, sess.GetRtt(), kbps
}
