package server

import (
	"fmt"
	"net"
)

// ResolveListener attempts to bind to the preferred port. If the port is busy
// and portExplicit is false, it falls back to an OS-assigned available port.
// If portExplicit is true, it returns an error when the preferred port is unavailable.
func ResolveListener(preferredPort int, portExplicit bool) (net.Listener, int, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", preferredPort))
	if err == nil {
		addr := ln.Addr().(*net.TCPAddr)
		return ln, addr.Port, nil
	}

	if portExplicit {
		return nil, 0, fmt.Errorf("port %d is already in use", preferredPort)
	}

	// Fall back to OS-assigned port
	ln, err = net.Listen("tcp", ":0")
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find available port: %w", err)
	}
	addr := ln.Addr().(*net.TCPAddr)
	return ln, addr.Port, nil
}
