package diagnostics

import (
	"bufio"
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
)

type SystemSnapshot struct {
	OS            string
	Architecture  string
	Kernel        string
	KernelError   string
	BTFPresent    bool
	BPFFSMounted  bool
	BPFErrors     []string
	DefaultRoutes []string
	RouteError    string
}

type nativeSystemProbe struct{}

func (nativeSystemProbe) Snapshot(ctx context.Context) SystemSnapshot {
	snapshot := SystemSnapshot{OS: runtime.GOOS, Architecture: runtime.GOARCH}
	if err := ctx.Err(); err != nil {
		snapshot.KernelError = err.Error()
		return snapshot
	}
	kernel, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		snapshot.KernelError = fmt.Sprintf("读取内核版本: %v", err)
	} else {
		snapshot.Kernel = strings.TrimSpace(string(kernel))
	}
	if info, err := os.Stat("/sys/kernel/btf/vmlinux"); err == nil && info.Mode().IsRegular() {
		snapshot.BTFPresent = true
	} else if err != nil {
		snapshot.BPFErrors = append(snapshot.BPFErrors, fmt.Sprintf("读取 /sys/kernel/btf/vmlinux: %v", err))
	}
	mounted, err := bpffsMounted("/proc/self/mountinfo")
	if err != nil {
		snapshot.BPFErrors = append(snapshot.BPFErrors, fmt.Sprintf("读取 bpffs 挂载状态: %v", err))
	} else {
		snapshot.BPFFSMounted = mounted
	}
	routes, routeErr := defaultRoutes()
	snapshot.DefaultRoutes = routes
	if routeErr != nil {
		snapshot.RouteError = routeErr.Error()
	}
	return snapshot
}

func bpffsMounted(path string) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(string(content), "\n") {
		left, right, found := strings.Cut(line, " - ")
		if !found {
			continue
		}
		fields := strings.Fields(left)
		filesystem := strings.Fields(right)
		if len(fields) > 4 && len(filesystem) > 0 && fields[4] == "/sys/fs/bpf" && filesystem[0] == "bpf" {
			return true, nil
		}
	}
	return false, nil
}

func defaultRoutes() ([]string, error) {
	var routes []string
	var failures []string
	if ipv4, err := ipv4DefaultRoutes("/proc/net/route"); err == nil {
		routes = append(routes, ipv4...)
	} else {
		failures = append(failures, "IPv4: "+err.Error())
	}
	if ipv6, err := ipv6DefaultRoutes("/proc/net/ipv6_route"); err == nil {
		routes = append(routes, ipv6...)
	} else {
		failures = append(failures, "IPv6: "+err.Error())
	}
	if len(failures) == 2 {
		return nil, fmt.Errorf("%s", strings.Join(failures, "；"))
	}
	return routes, nil
}

func ipv4DefaultRoutes(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var routes []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 8 || fields[1] != "00000000" || fields[7] != "00000000" {
			continue
		}
		gateway := parseIPv4Hex(fields[2])
		if gateway == "0.0.0.0" || gateway == "" {
			routes = append(routes, "IPv4 default dev "+fields[0])
		} else {
			routes = append(routes, fmt.Sprintf("IPv4 default via %s dev %s", gateway, fields[0]))
		}
	}
	return routes, scanner.Err()
}

func parseIPv4Hex(value string) string {
	bytes, err := hex.DecodeString(value)
	if err != nil || len(bytes) != net.IPv4len {
		return ""
	}
	return net.IPv4(bytes[3], bytes[2], bytes[1], bytes[0]).String()
}

func ipv6DefaultRoutes(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var routes []string
	for _, line := range strings.Split(string(content), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 10 || strings.Trim(fields[0], "0") != "" || fields[1] != "00" {
			continue
		}
		gateway := parseIPv6Hex(fields[4])
		metric, _ := strconv.ParseUint(fields[5], 16, 32)
		detail := fmt.Sprintf("IPv6 default dev %s metric %d", fields[len(fields)-1], metric)
		if gateway != "" && gateway != "::" {
			detail = fmt.Sprintf("IPv6 default via %s dev %s metric %d", gateway, fields[len(fields)-1], metric)
		}
		routes = append(routes, detail)
	}
	return routes, nil
}

func parseIPv6Hex(value string) string {
	bytes, err := hex.DecodeString(value)
	if err != nil || len(bytes) != net.IPv6len {
		return ""
	}
	return net.IP(bytes).String()
}
