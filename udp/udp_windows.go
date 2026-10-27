//go:build !e2e_testing

package udp

import (
	"fmt"
	"log/slog"
	"net"
	"syscall"
)

func NewListener(l *slog.Logger, s Settings) (Conn, error) {
	if s.Multi {
		//NOTE: Technically we can support it with RIO but it wouldn't be at the socket level
		// The udp stack would need to be reworked to hide away the implementation differences between
		// Windows and Linux
		return nil, fmt.Errorf("multiple udp listeners not supported on windows")
	}

	// Windows RIO opens its own socket and cannot honor net.ListenConfig.
	// When an embedder supplies one, use the standard UDP backend instead.
	if s.ListenConfig != nil {
		conn, err := NewGenericListener(l, s)
		if err != nil {
			return nil, err
		}
		return wrapWithWDFBypass(l, conn), nil
	}

	var conn Conn
	rc, err := NewRIOListener(l, s.Listen.Addr(), int(s.Listen.Port()))
	if err == nil {
		conn = rc
	} else {
		l.Error("Falling back to standard udp sockets", "error", err)
		conn, err = NewGenericListener(l, s)
		if err != nil {
			return nil, err
		}
	}
	return wrapWithWDFBypass(l, conn), nil
}

func NewListenConfig(multi bool, options ...func(*net.ListenConfig)) net.ListenConfig {
	var lc net.ListenConfig
	for _, option := range options {
		if option != nil {
			option(&lc)
		}
	}
	if multi {
		lc.Control = func(string, string, syscall.RawConn) error {
			// Windows does not support multiple UDP listeners for this backend.
			return fmt.Errorf("multiple udp listeners not supported on windows")
		}
	}
	return lc
}

func (u *GenericConn) Rebind() error {
	return nil
}
