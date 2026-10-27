//go:build !e2e_testing

package udp

import (
	"errors"
	"io"
	"log/slog"
	"net"
	"net/netip"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListenConfigOption(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	addr := netip.MustParseAddrPort("127.0.0.1:0")

	t.Run("control invoked before bind", func(t *testing.T) {
		calls := 0
		conn, err := NewListener(logger, Settings{
			Listen:       addr,
			Batch:        1,
			ListenConfig: func(lc *net.ListenConfig) {
				lc.Control = func(network, address string, c syscall.RawConn) error {
					calls++
					return c.Control(func(fd uintptr) { require.NotZero(t, fd) })
				}
			},
		})
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, conn.Close()) })
		require.Equal(t, 1, calls)
	})

	t.Run("control failure aborts listener", func(t *testing.T) {
		expected := errors.New("underlay socket protection failed")
		calls := 0
		conn, err := NewListener(logger, Settings{
			Listen:       addr,
			Batch:        1,
			ListenConfig: func(lc *net.ListenConfig) {
				lc.Control = func(string, string, syscall.RawConn) error {
					calls++
					return expected
				}
			},
		})
		require.Nil(t, conn)
		require.ErrorIs(t, err, expected)
		require.Equal(t, 1, calls)
	})
}
