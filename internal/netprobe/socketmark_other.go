//go:build !linux

package netprobe

import (
	"context"
	"net"
)

// 面板发行包只面向 Linux;其他平台保留普通拨号仅用于本地开发与单元测试。
func markedDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, network, address)
}
