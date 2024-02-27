package main

import (
	"crypto/tls"
	"log"
	"net/http"

	"github.com/lucas-clemente/quic-go/h2quic"
)

func main() {
	// Define the root directory you want to serve
	rootDir := "/var/www/html/tos_4sec_full/4K_dataset/4_sec/x264/bbb/DASH_Files/full/"

	// Create a file server handler that serves files from the specified directory
	fileServer := http.FileServer(http.Dir(rootDir))

	// Setup the HTTP muxer
	mux := http.NewServeMux()

	// Use StripPrefix to remove the leading part of the request path.
	// This makes it so /path/to/file will fetch /var/www/.../full/path/to/file
	mux.Handle("/", http.StripPrefix("/", fileServer))

	// Load the server TLS configuration
	tlsConfig, err := loadTLSConfig()
	if err != nil {
		log.Fatalf("Failed to load TLS configuration: %v", err)
	}

	// Create the HTTP3 server
	server := h2quic.Server{
		Server: &http.Server{
			Addr:      ":4242",
			Handler:   mux,
			TLSConfig: tlsConfig,
		},
	}

	log.Println("Starting QUIC server on https://localhost:4242")
	if err := server.ListenAndServeTLS("../godash/http/certs/cert.pem", "../godash/http/certs/key.pem"); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}

func loadTLSConfig() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair("../godash/http/certs/cert.pem", "../godash/http/certs/key.pem")
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h3", "quic"},
	}, nil
}
