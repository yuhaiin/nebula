//go:build linux && !android && !e2e_testing

package udp

import (
	"errors"
	"io"
	"log/slog"
	"net"
	"net/netip"
	"syscall"
	"testing"
	"time"

	"github.com/slackhq/nebula/config"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestListenConfigMarkSurvivesConfigLoad(t *testing.T) {
	const mark = 0x500
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	conn, err := NewListener(logger, Settings{
		Listen:       netip.MustParseAddrPort("127.0.0.1:0"),
		Batch:        1,
		ListenConfig: func(lc *net.ListenConfig) {
			lc.Control = func(_, _ string, c syscall.RawConn) error {
				var optionErr error
				if err := c.Control(func(fd uintptr) {
					optionErr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_MARK, mark)
				}); err != nil {
					return err
				}
				return optionErr
			}
		},
	})
	if errors.Is(err, unix.EPERM) || errors.Is(err, unix.EACCES) {
		t.Skipf("setting SO_MARK unavailable in test environment: %v", err)
	}
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	c := config.NewC(logger)
	require.NoError(t, c.LoadString("listen:\n  port: 0\n"))
	conn.ReloadConfig(c, nil)
	got, err := conn.(*StdConn).GetSoMark()
	require.NoError(t, err)
	require.Equal(t, mark, got)
}

func TestListenConfigPreservesReusePort(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	calls := 0
	opt := func(lc *net.ListenConfig) {
		lc.Control = func(_, _ string, _ syscall.RawConn) error {
			calls++
			return nil
		}
	}
	first, err := NewListener(logger, Settings{
		Listen:       netip.MustParseAddrPort("127.0.0.1:0"),
		Multi:        true,
		Batch:        1,
		ListenConfig: opt,
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, first.Close()) })
	local, err := first.LocalAddr()
	require.NoError(t, err)
	second, err := NewListener(logger, Settings{
		Listen:       local,
		Multi:        true,
		Batch:        1,
		ListenConfig: opt,
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, second.Close()) })
	require.Equal(t, 2, calls)
}

func TestConfiguredListenerUsesNativeUDPWriter(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	conn, err := NewListener(logger, Settings{
		Listen:       netip.MustParseAddrPort("127.0.0.1:0"),
		Batch:        1,
		ListenConfig: func(*net.ListenConfig) {},
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	receiver, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, receiver.Close()) })
	require.NoError(t, receiver.SetReadDeadline(time.Now().Add(3*time.Second)))

	addr, err := netip.ParseAddrPort(receiver.LocalAddr().String())
	require.NoError(t, err)
	require.NoError(t, conn.WriteTo([]byte("hello"), addr))
	buf := make([]byte, 32)
	n, _, err := receiver.ReadFromUDP(buf)
	require.NoError(t, err)
	require.Equal(t, "hello", string(buf[:n]))
}
