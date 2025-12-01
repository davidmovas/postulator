package proxy

import (
	"context"
	"fmt"
	"net"
	"time"
)

var torPorts = []int{9050, 9150}

type TorDetector struct {
	timeout time.Duration
}

func NewTorDetector() *TorDetector {
	return &TorDetector{
		timeout: 2 * time.Second,
	}
}

func (d *TorDetector) Detect(ctx context.Context) *TorDetectionResult {
	for _, port := range torPorts {
		if d.checkPort(ctx, port) {
			serviceType := "tor-service"
			if port == 9150 {
				serviceType = "tor-browser"
			}
			return &TorDetectionResult{
				Found:       true,
				Port:        port,
				ServiceType: serviceType,
			}
		}
	}

	return &TorDetectionResult{
		Found: false,
	}
}

func (d *TorDetector) checkPort(ctx context.Context, port int) bool {
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	dialer := &net.Dialer{
		Timeout: d.timeout,
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return false
	}
	conn.Close()

	return d.verifySocks5(ctx, addr)
}

func (d *TorDetector) verifySocks5(ctx context.Context, addr string) bool {
	dialer := &net.Dialer{
		Timeout: d.timeout,
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return false
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(d.timeout))

	_, err = conn.Write([]byte{0x05, 0x01, 0x00})
	if err != nil {
		return false
	}

	response := make([]byte, 2)
	_, err = conn.Read(response)
	if err != nil {
		return false
	}

	return response[0] == 0x05 && response[1] == 0x00
}

func (d *TorDetector) CheckTorConnectivity(ctx context.Context, port int) bool {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	return d.verifySocks5(ctx, addr)
}
