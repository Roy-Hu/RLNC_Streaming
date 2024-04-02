package xnc

import (
	"crypto/tls"
	"fmt"
)

func SpiltFile(filebytes []byte, chunkSize int) [][]byte {
	chunks := make([][]byte, 0)
	for i := 0; i < len(filebytes); i += chunkSize {
		end := Min(i+chunkSize, i+len(filebytes[i:]))
		chunkBytes := filebytes[i:end]

		// padding chunkbytes to chunk size
		if len(chunkBytes) < chunkSize {
			chunkBytes = append(chunkBytes, make([]byte, chunkSize-len(chunkBytes))...)
		}

		chunks = append(chunks, chunkBytes)
	}

	return chunks
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func GenerateTLSConfig() *tls.Config {
	cert, err := tls.LoadX509KeyPair("../godash/http/certs/cert.pem", "../godash/http/certs/key.pem")
	if err != nil {
		fmt.Printf("TLS config err: %v", err)

		return nil
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
}
