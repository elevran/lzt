package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"os"
	"syscall"
)

func loadTLSConfig(caPath, certPath, keyPath string) (*tls.Config, error) {
	// Load certificate and key
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load workload certificate and key (%s/%s): %w",
			certPath, keyPath, err)
	}

	// Load CA certificate
	caCert, err := os.ReadFile(caPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate (%s): %w", caPath, err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Create TLS configuration
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}, nil
}

// server side handling by the supervisor - TLS handshake
// NOTE: monitored process is not continued on errors!
func tlsServerSide(supervisor, monitored int, conn net.Conn, tlsConfig *tls.Config) error {
	log.Printf("server side supervisor (pid %d) hijacked %s -> %s from pid %d", supervisor,
		conn.RemoteAddr().String(), conn.LocalAddr().String(), monitored)

	defer conn.Close()

	tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert // enforce mTLS client authentication
	tlsConfig.ClientCAs = tlsConfig.RootCAs

	tlsConn := tls.Server(conn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		return err
	}

	// Extract and print peer certificate
	state := tlsConn.ConnectionState()
	if len(state.PeerCertificates) > 0 {
		peerCert := state.PeerCertificates[0]
		log.Printf("supervisor (pid %d) TLS handshake complete with Subject: %s\n",
			supervisor, peerCert.Subject)
	} else {
		log.Printf("supervisor (pid %d) TLS handshake complete without peer\n", supervisor)
	}

	if err := enableKTLS(conn); err != nil {
		return fmt.Errorf("supervisor (pid %d) failed to enable kTLS for server: %w",
			supervisor, err)
	} else {
		log.Printf("supervisor (pid %d) successfully enabled kTLS on server", supervisor)
	}
	return syscall.Kill(monitored, syscall.SIGCONT)
}

// client side handling by the supervisor - TLS handshake
// NOTE: monitored process is not continued on errors!
func tlsClientSide(supervisor, monitored int, conn net.Conn, tlsConfig *tls.Config) error {
	log.Printf("client side supervisor (pid %d) hijacked %s -> %s from pid %d", supervisor,
		conn.LocalAddr().String(), conn.RemoteAddr().String(), monitored)

	defer conn.Close()

	tlsConfig.ServerName = "server.lztdemo" // need to map IP to server name?
	tlsConn := tls.Client(conn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		return err
	}

	// Extract and print peer certificate
	state := tlsConn.ConnectionState()
	if len(state.PeerCertificates) > 0 {
		peerCert := state.PeerCertificates[0]
		log.Printf("supervisor (pid %d) TLS handshake complete with Subject: %s\n",
			supervisor, peerCert.Subject)
	} else {
		log.Printf("supervisor (pid %d) TLS handshake complete without peer\n", supervisor)
	}

	if err := enableKTLS(conn); err != nil {
		return fmt.Errorf("supervisor (pid %d) failed to enable kTLS for client: %w",
			supervisor, err)
	} else {
		log.Printf("supervisor (pid %d) successfully enabled kTLS on client", supervisor)
	}
	return syscall.Kill(monitored, syscall.SIGCONT)
}
