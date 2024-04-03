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
	var chunk []byte

	if encode {
		initType = TYPE_INIT_ENC
		frameSize = FRAMESIZE_ENC
	} else {
		initType = TYPE_INIT
		frameSize = FRAMESIZE
		chunk = make([]byte, 0, CHUNKSIZE)
	}

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

	var decoders []*full.FullRLNCDecoder

	for {
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

		if xncD.Type == TYPE_END {
			fmt.Printf("[Client] Received END packet\n")
			break
		}

		if encode {
			if decoders == nil {
				decoders = make([]*full.FullRLNCDecoder, xncD.ChunkNum)
				for i := 0; i < xncD.ChunkNum; i++ {
					decoders[i] = full.NewFullRLNCDecoder(PIECECNT)
				}
			}

			pieceD := &kodr.CodedPiece{
				Vector: xncD.Vector,
				Piece:  xncD.Piece,
			}

			if err := decoders[xncD.ChunkId].AddPiece(pieceD); err != nil {
				if errors.Is(err, kodr.ErrAllUsefulPiecesReceived) {
					// fmt.Printf("[Client] All useful pieces received\n")
					continue
				} else {
					fmt.Printf("[Client] Error adding pieces: %v", err)
					return nil, sess.GetRtt(), 0
				}
			}
			// loss debug
			// fmt.Printf("[Client] Chunk %d, recv %d, need %d\n", xncD.ChunkId, decoders[xncD.ChunkId].GetRecv(), decoders[xncD.ChunkId].GetExpt())

			if decoders[xncD.ChunkId].IsDecoded() {
				recvfile, err := GetFile(decoders[xncD.ChunkId])
				if err != nil {
					fmt.Printf("[Client] Error geting file: %v", err)
					return nil, sess.GetRtt(), 0
				}

				rFile = append(rFile, recvfile[:xncD.ChunkSize]...)

				if xncD.ChunkId == xncD.ChunkNum-1 {
					fmt.Printf("[Client] Finished decoding file\n")
					break
				}
			}
		} else {
			chunk = append(chunk, xncD.Piece...)
			if len(chunk) == CHUNKSIZE {
				// fmt.Printf("[Client] Received chunk %v\n", xncD.ChunkId)

				rFile = append(rFile, chunk[:xncD.ChunkSize]...)
				chunk = make([]byte, 0, CHUNKSIZE)

				if xncD.ChunkId == xncD.ChunkNum-1 {
					fmt.Printf("[Client] Finished decoding file\n")
					break
				}
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
