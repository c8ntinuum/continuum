package network

import (
	"fmt"
	"net"
)

func init() {
	const poolSize = 200
	for i := 0; i < poolSize; i++ {
		port, err := getFreePort()
		if err != nil {
			continue
		}
		portPool <- port
	}
	if len(portPool) == 0 {
		panic("failed to initialize test port pool")
	}
}

func getFreePort() (string, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer l.Close()

	addr, ok := l.Addr().(*net.TCPAddr)
	if !ok {
		return "", fmt.Errorf("unexpected listener address type: %T", l.Addr())
	}
	return fmt.Sprintf("%d", addr.Port), nil
}
