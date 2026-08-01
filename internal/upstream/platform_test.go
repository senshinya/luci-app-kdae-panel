package upstream

import (
	"slices"
	"strings"
	"testing"
)

func flagSet(names ...string) func() map[string]bool {
	return func() map[string]bool {
		flags := make(map[string]bool, len(names))
		for _, name := range names {
			flags[name] = true
		}
		return flags
	}
}

func TestDetectPlatformPicksBestX86Variant(t *testing.T) {
	v2 := append([]string(nil), x86V2Flags...)
	v3 := append(append([]string(nil), x86V2Flags...), x86V3Flags...)
	missingV3 := slices.DeleteFunc(append([]string(nil), v3...), func(flag string) bool { return flag == "bmi2" })
	cases := []struct {
		name      string
		flags     []string
		want      string
		fallbacks []string
	}{
		{"完整 v3", v3, "x86_64_v3_avx2", []string{"x86_64_v2_sse", "x86_64"}},
		{"v3 缺少 bmi2", missingV3, "x86_64_v2_sse", []string{"x86_64"}},
		{"只有 avx2", []string{"avx2"}, "x86_64", nil},
		{"完整 v2", v2, "x86_64_v2_sse", []string{"x86_64"}},
		{"只有 sse4.2", []string{"sse4_2"}, "x86_64", nil},
		{"老 CPU", []string{"fpu"}, "x86_64", nil},
		{"读不到 cpuinfo", nil, "x86_64", nil},
	}
	for _, testCase := range cases {
		platform, err := detectPlatform("amd64", flagSet(testCase.flags...))
		if err != nil {
			t.Fatalf("%s: %v", testCase.name, err)
		}
		if platform.Name != testCase.want {
			t.Fatalf("%s: 首选 = %q，期望 %q", testCase.name, platform.Name, testCase.want)
		}
		if platform.Architecture != "x86_64" {
			t.Fatalf("%s: CPU 架构 = %q，期望 x86_64", testCase.name, platform.Architecture)
		}
		if !slices.Equal(platform.Fallbacks, testCase.fallbacks) {
			t.Fatalf("%s: 回退 = %v，期望 %v", testCase.name, platform.Fallbacks, testCase.fallbacks)
		}
		// 无论选中哪一档，最后一个候选都必须是最保守的基础变体
		candidates := platform.Candidates()
		if candidates[len(candidates)-1] != "x86_64" {
			t.Fatalf("%s: 最终回退 = %q，期望 x86_64", testCase.name, candidates[len(candidates)-1])
		}
	}
}

func TestDetectPlatformArmVariants(t *testing.T) {
	cases := []struct {
		flags []string
		want  string
	}{
		{[]string{"neon"}, "armv7"},
		{[]string{"vfpv3"}, "armv7"},
		{[]string{"vfp"}, "armv6"},
		{nil, "armv5"},
	}
	for _, testCase := range cases {
		platform, err := detectPlatform("arm", flagSet(testCase.flags...))
		if err != nil {
			t.Fatal(err)
		}
		if platform.Name != testCase.want {
			t.Fatalf("flags=%v 首选 = %q，期望 %q", testCase.flags, platform.Name, testCase.want)
		}
		if platform.Architecture != "arm" {
			t.Fatalf("flags=%v CPU 架构 = %q，期望 arm", testCase.flags, platform.Architecture)
		}
		if candidates := platform.Candidates(); candidates[len(candidates)-1] != "armv5" {
			t.Fatalf("flags=%v 最终回退 = %q，期望 armv5", testCase.flags, candidates[len(candidates)-1])
		}
	}
}

func TestDetectPlatformSimpleArchitectures(t *testing.T) {
	expected := map[string]Platform{
		"arm64":    {Architecture: "arm64", Name: "arm64"},
		"386":      {Architecture: "x86_32", Name: "x86_32"},
		"riscv64":  {Architecture: "riscv64", Name: "riscv64"},
		"loong64":  {Architecture: "loongarch64", Name: "loongarch64"},
		"s390x":    {Architecture: "s390x", Name: "s390x"},
		"ppc64":    {Architecture: "powerpc64", Name: "powerpc64"},
		"ppc64le":  {Architecture: "powerpc64le", Name: "powerpc64le"},
		"mips":     {Architecture: "mips32", Name: "mips32"},
		"mipsle":   {Architecture: "mips32le", Name: "mips32le"},
		"mips64":   {Architecture: "mips64", Name: "mips64"},
		"mips64le": {Architecture: "mips64le", Name: "mips64le"},
	}
	for goarch, want := range expected {
		platform, err := detectPlatform(goarch, flagSet())
		if err != nil {
			t.Fatalf("%s: %v", goarch, err)
		}
		if platform.Architecture != want.Architecture || platform.Name != want.Name || len(platform.Fallbacks) != 0 {
			t.Fatalf("%s: 得到 %+v，期望 %+v 且无回退", goarch, platform, want)
		}
	}
}

func TestDetectPlatformRejectsUnknownArchitecture(t *testing.T) {
	_, err := detectPlatform("sparc64", flagSet())
	if err == nil || !strings.Contains(err.Error(), "sparc64") {
		t.Fatalf("未知架构应报错并指明架构名，得到 %v", err)
	}
}

func TestParseCPUFlagsUsesFeaturesSharedByEveryCPU(t *testing.T) {
	flags := parseCPUFlags(`processor: 0
flags: fpu sse4_2 avx2 bmi2
processor: 1
flags: fpu sse4_2 avx2
`)
	for _, shared := range []string{"fpu", "sse4_2", "avx2"} {
		if !flags[shared] {
			t.Fatalf("共同特性 %q 丢失: %v", shared, flags)
		}
	}
	if flags["bmi2"] {
		t.Fatalf("只在部分 CPU 上存在的特性不应被采用: %v", flags)
	}
}

func TestParseCPUFlagsWithoutFeatureLineReturnsEmptySet(t *testing.T) {
	if flags := parseCPUFlags("processor: 0\nmodel name: unknown\n"); len(flags) != 0 {
		t.Fatalf("没有特性行时应返回空集: %v", flags)
	}
}

func TestAssetName(t *testing.T) {
	if got := AssetName("x86_64_v3_avx2"); got != "dae-linux-x86_64_v3_avx2.zip" {
		t.Fatalf("资产名 = %q", got)
	}
}
