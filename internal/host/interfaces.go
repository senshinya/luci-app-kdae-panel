package host

import (
	"context"
	"net"
	"sort"
)

// NetworkInterface 是面板可供用户选择的本机网络接口。
type NetworkInterface struct {
	Name      string   `json:"name"`
	Addresses []string `json:"addresses,omitempty"`
}

// interfaceLister 提供与 init 系统无关的本机网卡枚举。
//
// Manager 变成接口后这个方法不能再挂在某个具体后端上，而它的实现只用
// net.Interfaces()，两个后端没有任何理由各写一份。做成零字段结构体由
// 两个后端各内嵌一次，方法集自动带上。
type interfaceLister struct{}

// Interfaces 枚举本机接口及其地址。单个接口读取地址失败时仍保留接口名，
// 让尚未分配地址或状态正在变化的接口也能出现在配置选择器中。
func (interfaceLister) Interfaces(ctx context.Context) ([]NetworkInterface, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	result := make([]NetworkInterface, 0, len(interfaces))
	for _, networkInterface := range interfaces {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		item := NetworkInterface{Name: networkInterface.Name}
		addresses, err := networkInterface.Addrs()
		if err == nil {
			seen := make(map[string]struct{}, len(addresses))
			for _, address := range addresses {
				value := address.String()
				if _, exists := seen[value]; exists {
					continue
				}
				seen[value] = struct{}{}
				item.Addresses = append(item.Addresses, value)
			}
			sort.Strings(item.Addresses)
		}
		result = append(result, item)
	}
	sort.Slice(result, func(left, right int) bool {
		return result[left].Name < result[right].Name
	})
	return result, nil
}
