//go:build linux

package netprobe

import (
	"context"
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

// daeBypassMark 是 dae 从 v1 系列起就在 WAN egress 中保留的绕行位;
// 新版 dae 在 so_mark_from_dae 为 0 时也会把它作为内部实际值。
const daeBypassMark = 0x100

var markedDialer = net.Dialer{
	Control: func(_, _ string, raw syscall.RawConn) error {
		var setErr error
		if err := raw.Control(func(fd uintptr) {
			setErr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_MARK, daeBypassMark)
		}); err != nil {
			return err
		}
		return setErr
	},
}

func markedDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return markedDialer.DialContext(ctx, network, address)
}
