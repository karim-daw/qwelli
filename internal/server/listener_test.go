package server

import (
	"net"
	"testing"
)

func TestResolveListener_PreferredPortAvailable(t *testing.T) {
	// Port 0 asks the OS for any available port — should always succeed
	ln, port, err := ResolveListener(0, false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer ln.Close()

	if port == 0 {
		t.Fatal("expected non-zero port")
	}
}

func TestResolveListener_FallbackWhenBusy(t *testing.T) {
	// Occupy a port first
	occupied, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to occupy port: %v", err)
	}
	defer occupied.Close()
	busyPort := occupied.Addr().(*net.TCPAddr).Port

	// Try the busy port with portExplicit=false — should fall back
	ln, port, err := ResolveListener(busyPort, false)
	if err != nil {
		t.Fatalf("expected fallback to succeed, got: %v", err)
	}
	defer ln.Close()

	if port == busyPort {
		t.Fatalf("expected different port, got same busy port %d", busyPort)
	}
	if port == 0 {
		t.Fatal("expected non-zero fallback port")
	}
}

func TestResolveListener_ExplicitPortBusy(t *testing.T) {
	// Occupy a port first
	occupied, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to occupy port: %v", err)
	}
	defer occupied.Close()
	busyPort := occupied.Addr().(*net.TCPAddr).Port

	// Try the busy port with portExplicit=true — should fail
	ln, _, err := ResolveListener(busyPort, true)
	if ln != nil {
		ln.Close()
		t.Fatal("expected no listener when explicit port is busy")
	}
	if err == nil {
		t.Fatal("expected error when explicit port is busy")
	}
}
