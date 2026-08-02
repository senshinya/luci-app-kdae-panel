//go:build !linux

package netprobe

import (
	"context"
	"errors"
	"net"
)

func probeICMP(context.Context, net.IP) (float64, error) {
	return 0, errors.New("当前平台不支持 ICMP 网络延迟探测")
}
