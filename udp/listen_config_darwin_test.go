//go:build darwin && !e2e_testing

package udp

import (
	"io"
	"log/slog"
	"net"
	"net/netip"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRebindReappliesListenConfigControl(t *testing.T) {
	calls := 0
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	conn, err := NewListener(logger, Settings{
		Listen:       netip.MustParseAddrPort("127.0.0.1:0"),
		Batch:        1,
		ListenConfig: func(lc *net.ListenConfig) {
			lc.Control = func(_, _ string, _ syscall.RawConn) error {
				calls++
				return nil
			}
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	require.Equal(t, 1, calls)
	require.NoError(t, conn.Rebind())
	require.Equal(t, 2, calls)
}
