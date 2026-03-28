package monitor

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestTCPCheck_Healthy(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	c := NewTCPCheck("test-tcp", ln.Addr().String(), 2*time.Second)
	result := c.Check(context.Background())

	if result.Status != StatusHealthy {
		t.Errorf("expected healthy, got %s (err: %s)", result.Status, result.Error)
	}
}

func TestTCPCheck_Down(t *testing.T) {
	c := NewTCPCheck("test-tcp", "127.0.0.1:1", 500*time.Millisecond)
	result := c.Check(context.Background())

	if result.Status != StatusDown {
		t.Errorf("expected down, got %s", result.Status)
	}
}
