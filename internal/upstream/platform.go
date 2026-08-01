package upstream

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// Platform 描述当前主机对应的 dae 发布资产标识。
// dae 的资产按 dae-linux-<平台>.zip 命名,x86_64 与 arm 还有按指令集细分的变体,
// 选错会直接以非法指令崩溃,因此这里按实际 CPU 特性挑选而不是只看 GOARCH。
type Platform struct {
	// Architecture 是不含优化等级的 CPU 架构,如 x86_64。
	Architecture string
	// Name 首选资产标识,如 x86_64_v3_avx2。
	Name string
	// Fallbacks 首选不存在时按顺序回退,末尾一定是最保守的基础变体。
	Fallbacks []string
}

var (
	// x86-64-v2 除基线指令外要求这些扩展。Linux 在 /proc/cpuinfo 里把
	// SSE3 写作 pni,把 CMPXCHG16B 写作 cx16。
	x86V2Flags = []string{
		"cx16", "lahf_lm", "popcnt", "pni", "ssse3", "sse4_1", "sse4_2",
	}
	// v3 必须在完整 v2 之上再具备这些扩展。只检查 avx2 不够：虚拟机
	// 可能选择性透传指令位,误选后会在运行时以非法指令退出。
	x86V3Flags = []string{
		"abm", "avx", "avx2", "bmi1", "bmi2", "f16c", "fma", "movbe", "xsave",
	}
)

// Candidates 返回按优先级排列的资产标识。
func (p Platform) Candidates() []string {
	return append([]string{p.Name}, p.Fallbacks...)
}

// DetectPlatform 依据 GOARCH 与 CPU 特性推断资产标识。
func DetectPlatform() (Platform, error) {
	return detectPlatform(runtime.GOARCH, readCPUFlags)
}

func detectPlatform(goarch string, flags func() map[string]bool) (Platform, error) {
	switch goarch {
	case "amd64":
		// 上游同时提供基础版与两个优化版,优先用主机支持的最高档。
		cpu := flags()
		switch {
		case hasAll(cpu, x86V2Flags) && hasAll(cpu, x86V3Flags):
			return Platform{Architecture: "x86_64", Name: "x86_64_v3_avx2", Fallbacks: []string{"x86_64_v2_sse", "x86_64"}}, nil
		case hasAll(cpu, x86V2Flags):
			return Platform{Architecture: "x86_64", Name: "x86_64_v2_sse", Fallbacks: []string{"x86_64"}}, nil
		default:
			return Platform{Architecture: "x86_64", Name: "x86_64"}, nil
		}
	case "386":
		return Platform{Architecture: "x86_32", Name: "x86_32"}, nil
	case "arm64":
		return Platform{Architecture: "arm64", Name: "arm64"}, nil
	case "arm":
		// GOARM 在编译期固定,运行期改看 CPU 特性:
		// neon/vfpv3 对应 v7,vfp 对应 v6,其余按 v5 处理。
		cpu := flags()
		switch {
		case cpu["neon"] || cpu["vfpv3"]:
			return Platform{Architecture: "arm", Name: "armv7", Fallbacks: []string{"armv6", "armv5"}}, nil
		case cpu["vfp"]:
			return Platform{Architecture: "arm", Name: "armv6", Fallbacks: []string{"armv5"}}, nil
		default:
			return Platform{Architecture: "arm", Name: "armv5"}, nil
		}
	case "riscv64":
		return Platform{Architecture: "riscv64", Name: "riscv64"}, nil
	case "loong64":
		return Platform{Architecture: "loongarch64", Name: "loongarch64"}, nil
	case "s390x":
		return Platform{Architecture: "s390x", Name: "s390x"}, nil
	case "ppc64":
		return Platform{Architecture: "powerpc64", Name: "powerpc64"}, nil
	case "ppc64le":
		return Platform{Architecture: "powerpc64le", Name: "powerpc64le"}, nil
	case "mips":
		return Platform{Architecture: "mips32", Name: "mips32"}, nil
	case "mipsle":
		return Platform{Architecture: "mips32le", Name: "mips32le"}, nil
	case "mips64":
		return Platform{Architecture: "mips64", Name: "mips64"}, nil
	case "mips64le":
		return Platform{Architecture: "mips64le", Name: "mips64le"}, nil
	default:
		return Platform{}, fmt.Errorf("尚不支持的 CPU 架构 %q", goarch)
	}
}

func hasAll(flags map[string]bool, required []string) bool {
	for _, flag := range required {
		if !flags[flag] {
			return false
		}
	}
	return true
}

// readCPUFlags 解析 /proc/cpuinfo 的 flags/Features 行。
// 读不到时返回空集合,调用方因此退回最保守的基础变体。
func readCPUFlags() map[string]bool {
	content, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return make(map[string]bool)
	}
	return parseCPUFlags(string(content))
}

// parseCPUFlags 取所有逻辑 CPU 共同支持的特性，而不是做并集。
// 调度器可能把进程迁到任意核心，只在部分 vCPU 上可用的指令不能用于选择构建。
func parseCPUFlags(content string) map[string]bool {
	var common map[string]bool
	for _, line := range strings.Split(content, "\n") {
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		switch strings.TrimSpace(key) {
		case "flags", "Features":
			current := make(map[string]bool)
			for _, flag := range strings.Fields(value) {
				current[flag] = true
			}
			if common == nil {
				common = current
				continue
			}
			for flag := range common {
				if !current[flag] {
					delete(common, flag)
				}
			}
		}
	}
	if common == nil {
		return make(map[string]bool)
	}
	return common
}

// AssetName 返回资产文件名,如 dae-linux-x86_64.zip。
func AssetName(platform string) string {
	return "dae-linux-" + platform + ".zip"
}
