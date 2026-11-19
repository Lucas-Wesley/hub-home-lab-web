package main

import (
	"context"
	"net"
	"os"
	"syscall"
	"time"
)

// pingar manda um eco ICMP e devolve o tempo em ms (-1 sem resposta). Usa o socket ICMP sem
// privilégio (SOCK_DGRAM), que o Docker libera nos contêineres (net.ipv4.ping_group_range) e o
// macOS também, então o Hub não precisa de NET_RAW.
func pingar(ctx context.Context, endereco string) int {
	ip := net.ParseIP(endereco).To4()
	if ip == nil {
		return -1
	}
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_ICMP)
	if err != nil {
		return -1
	}
	f := os.NewFile(uintptr(fd), "icmp")
	c, err := net.FilePacketConn(f)
	f.Close()
	if err != nil {
		return -1
	}
	defer c.Close()

	prazo := time.Now().Add(esperaDoTeste)
	if d, ok := ctx.Deadline(); ok && d.Before(prazo) {
		prazo = d
	}
	c.SetDeadline(prazo)

	const seq = 1
	msg := []byte{8, 0, 0, 0, 0, 0, 0, seq, 'h', 'u', 'b', '-', 'v', 'i', 'g', 'i', 'a'}
	soma := checksum(msg)
	msg[2], msg[3] = byte(soma>>8), byte(soma)

	inicio := time.Now()
	if _, err := c.WriteTo(msg, &net.UDPAddr{IP: ip}); err != nil {
		return -1
	}
	buf := make([]byte, 1500)
	for {
		n, _, err := c.ReadFrom(buf)
		if err != nil {
			return -1
		}
		r := buf[:n]
		if len(r) > 20 && r[0]>>4 == 4 { // o macOS entrega com o cabeçalho IP
			r = r[int(r[0]&0x0f)*4:]
		}
		if len(r) >= 8 && r[0] == 0 && r[7] == seq {
			return max(1, int(time.Since(inicio).Milliseconds()))
		}
	}
}

func checksum(b []byte) uint16 {
	var s uint32
	for i := 0; i+1 < len(b); i += 2 {
		s += uint32(b[i])<<8 | uint32(b[i+1])
	}
	if len(b)%2 == 1 {
		s += uint32(b[len(b)-1]) << 8
	}
	for s>>16 != 0 {
		s = s&0xffff + s>>16
	}
	return ^uint16(s)
}
