//go:build !e2e_testing

package udp

// FreeBSD support is primarily implemented in udp_generic, besides NewListenConfig

import (
	"fmt"
	"log/slog"
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

func NewListener(l *slog.Logger, s Settings) (Conn, error) {
	return NewGenericListener(l, s)
}

func NewListenConfig(multi bool, options ...func(*net.ListenConfig)) net.ListenConfig {
	var lc net.ListenConfig
	for _, option := range options {
		if option != nil {
			option(&lc)
		}
	}
	if !multi {
		return lc
	}

	control := lc.Control
	lc.Control = func(network, address string, c syscall.RawConn) error {
		var optionErr error
		if err := c.Control(func(fd uintptr) {
			optionErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, unix.SO_REUSEPORT, 1)
		}); err != nil {
			return err
		}
		if optionErr != nil {
			return fmt.Errorf("SO_REUSEPORT failed: %w", optionErr)
		}
		if control != nil {
			return control(network, address, c)
		}
		return nil
	}
	return lc
}

func (u *GenericConn) Rebind() error {
	return nil
}
