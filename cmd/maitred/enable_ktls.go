package main

import (
	"fmt"
	"net"
	"syscall"
	"unsafe"
)

// Constants for setting socket options
const (
	TCP_ULP = 31
	SOL_TLS = 282
	TLS_TX  = 1
	TLS_RX  = 2

	TLS_CIPHER_AES_GCM_128 = uint16(51)
)

// TLS12CryptoInfo represents the necessary TLS 1.2 information structure
type TLS12CryptoInfo struct {
	Version    uint16
	CipherType uint16
	IV         [8]byte  // IV size for AES
	Key        [16]byte // AES-128 key
	Salt       [4]byte  // AES-128 salt
	RecSeq     [8]byte  // AES-128 record sequence
}

// convertToBytes converts the TLS12CryptoInfo to a byte array
func (info *TLS12CryptoInfo) convertToBytes() []byte {
	return (*[unsafe.Sizeof(*info)]byte)(unsafe.Pointer(info))[:]
}

func enableKTLS(conn net.Conn) error {
	// Extract the file descriptor from the connection
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return fmt.Errorf("not a TCP connection")
	}

	file, err := tcpConn.File()
	if err != nil {
		return fmt.Errorf("failed to get TCP connection file: %w", err)
	}
	defer file.Close()

	fd := int(file.Fd())

	// Set TCP_ULP option to "tls"
	ulp := "tls\x00"
	if err := syscall.SetsockoptString(fd, syscall.IPPROTO_TCP, TCP_ULP, ulp); err != nil {
		return fmt.Errorf("failed to set TCP_ULP option: %w", err)
	}

	// Define the (fake) TLS parameters (leave real key, salt etc as all zeros and
	// hope/pray for the best)
	ktlsInfo := TLS12CryptoInfo{
		Version:    0x0303, // TLS 1.2
		CipherType: TLS_CIPHER_AES_GCM_128,
	}
	tlsInfoBytes := ktlsInfo.convertToBytes()

	level := syscall.SOL_SOCKET
	for _, optname := range []int{TLS_RX, TLS_TX} {
		_, _, errno := syscall.Syscall6(
			syscall.SYS_SETSOCKOPT,
			uintptr(fd),
			uintptr(level),
			uintptr(optname),
			uintptr(unsafe.Pointer(&tlsInfoBytes[0])),
			uintptr(len(tlsInfoBytes)),
			0,
		)

		if errno != 0 {
			return fmt.Errorf("error setting socket option %d: %v", optname, errno)
		}
	}
	return nil
}
