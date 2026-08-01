package upstream

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
)

// BinaryName 是发布包内 dae 可执行文件的名字。
const BinaryName = "dae"

const (
	// maxBinaryBytes 限制解压后的可执行文件大小,防御 zip 炸弹。
	maxBinaryBytes = 200 << 20
	// maxExtraBytes 限制随包物料的大小。geoip.dat 约 17MB、geosite.dat 约 10MB。
	maxExtraBytes = 64 << 20
	// maxZipEntries 限制条目数。归档内容完全由上游构建产物决定,
	// 声明的 UncompressedSize64 也是攻击者可控的,因此实际读入量另由
	// LimitReader 兜底,这里只挡住"条目多到离谱"的归档。
	maxZipEntries = 256
)

// Bundle 是发布包里首次安装用得上的全部物料。
//
// 官方发布包不只有可执行文件,还平铺着 systemd 单元与两个 geo 数据
// 文件。它们全都被同一个 sha256 覆盖,因此首次安装不需要引入任何新的下载源
// 或信任根——这比另外去取 geo 数据安全得多。
type Bundle struct {
	// Platform 是解出此二进制的实际资产变体。
	Platform string
	Binary   []byte
	// Unit 是发布包自带的 dae.service,可能不存在。
	Unit    []byte
	GeoIP   []byte
	GeoSite []byte
}

// FetchBundle 下载资产、比对 sha256 并取出其中全部可用物料。
// 校验不通过时返回错误且不产出任何内容——调用方据此保证只有可信字节进入后续流程。
func (r *Registry) FetchBundle(ctx context.Context, asset Asset) (Bundle, error) {
	payload, err := r.fetchAsset(ctx, asset)
	if err != nil {
		return Bundle{}, err
	}
	bundle, err := extractArchive(payload, asset.Nested, true)
	if err != nil {
		return Bundle{}, err
	}
	bundle.Platform = asset.Platform
	return bundle, nil
}

// FetchBinary 供已有 dae 的升级与切换使用。发布包仍须完整下载并校验，
// 但无需再解压首次安装才会用到的服务单元和几十兆 geo 数据。
func (r *Registry) FetchBinary(ctx context.Context, asset Asset) ([]byte, error) {
	payload, err := r.fetchAsset(ctx, asset)
	if err != nil {
		return nil, err
	}
	bundle, err := extractArchive(payload, asset.Nested, false)
	if err != nil {
		return nil, err
	}
	return bundle.Binary, nil
}

func (r *Registry) fetchAsset(ctx context.Context, asset Asset) ([]byte, error) {
	if asset.SHA256 == "" {
		return nil, errors.New("资产缺少校验和，拒绝下载")
	}
	limit := int64(MaxAssetBytes)
	if asset.Size > 0 && asset.Size < limit {
		// 已知大小时收紧上限,留一点余量容忍元数据差异。
		limit = asset.Size + (1 << 20)
	}
	payload, err := r.client.download(ctx, asset.URL, limit)
	if err != nil {
		return nil, err
	}
	if err := verifyDigest(payload, asset.SHA256); err != nil {
		return nil, err
	}
	return payload, nil
}

func verifyDigest(payload []byte, expected string) error {
	sum := sha256.Sum256(payload)
	actual := hex.EncodeToString(sum[:])
	if subtle.ConstantTimeCompare([]byte(actual), []byte(strings.ToLower(expected))) != 1 {
		return fmt.Errorf("校验和不匹配：期望 %s，实际 %s", expected, actual)
	}
	return nil
}

// extractBundle 从发布包中取出全部可用物料。
// 只有可执行文件是必需的，其余缺失时留空由调用方决定如何应对。
func extractBundle(payload []byte, nested bool) (Bundle, error) {
	return extractArchive(payload, nested, true)
}

// extractArchive 的 includeExtras 为假时只读取二进制条目。zip 目录仍会完整校验，
// 但不会为升级路径解压和分配首次安装物料。
func extractArchive(payload []byte, nested, includeExtras bool) (Bundle, error) {
	if nested {
		inner, err := extractInnerArchive(payload)
		switch {
		case err == nil:
			payload = inner
		case errors.Is(err, errNoInnerArchive):
		default:
			return Bundle{}, err
		}
	}
	reader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		return Bundle{}, fmt.Errorf("解析发布包: %w", err)
	}
	if len(reader.File) > maxZipEntries {
		return Bundle{}, fmt.Errorf("发布包条目数超过 %d 限制", maxZipEntries)
	}

	var bundle Bundle
	var binaryEntry *zip.File
	extras := map[string]*[]byte{
		"dae.service": &bundle.Unit,
		"geoip.dat":   &bundle.GeoIP,
		"geosite.dat": &bundle.GeoSite,
	}
	for _, file := range reader.File {
		if file.FileInfo().IsDir() || !file.Mode().IsRegular() {
			continue
		}
		// 只按基名匹配，条目路径从不参与落盘，天然免疫 zip 路径穿越。
		name := path.Base(file.Name)
		if isBinaryEntry(name) {
			if binaryEntry != nil {
				return Bundle{}, errors.New("发布包中有多个 dae 条目，无法判断该用哪个")
			}
			binaryEntry = file
			continue
		}
		if target, ok := extras[name]; includeExtras && ok && *target == nil {
			content, err := readZipEntry(file, maxExtraBytes)
			if err != nil {
				return Bundle{}, err
			}
			*target = content
		}
	}
	if binaryEntry == nil {
		return Bundle{}, errors.New("发布包中没有找到 dae 可执行文件")
	}
	if bundle.Binary, err = readZipEntry(binaryEntry, maxBinaryBytes); err != nil {
		return Bundle{}, err
	}
	return bundle, nil
}

// extractBinary 只取发布包里的可执行文件，供测试直接调用。
// 生产路径的 FetchBinary 也复用 extractArchive，解包逻辑不会分叉。
func extractBinary(payload []byte, nested bool) ([]byte, error) {
	bundle, err := extractArchive(payload, nested, false)
	if err != nil {
		return nil, err
	}
	return bundle.Binary, nil
}

// isBinaryEntry 判断 zip 条目是否是 dae 可执行文件。
//
// 官方发布包里它并不叫 dae，而是按平台命名，如 dae-linux-x86_64
// （release.yml 用 install -D pkgdir/usr/bin/dae ./zip/dae-$ASSET_NAME 打包）。
// 同一个包里还有 dae.service、example.dae、empty.dae 与两个 .dat，必须排除。
func isBinaryEntry(name string) bool {
	if name != BinaryName && !strings.HasPrefix(name, BinaryName+"-") {
		return false
	}
	// 带扩展名的都是随包附带的其它物料，不是可执行文件。
	return path.Ext(name) == ""
}

// extractInnerArchive 取出外层 zip 里唯一的那个 zip。
// 嵌套深度硬编码为两层,不写通用递归,避免可控的递归深度。
func extractInnerArchive(payload []byte) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		return nil, fmt.Errorf("解析构建产物: %w", err)
	}
	if len(reader.File) > maxZipEntries {
		return nil, fmt.Errorf("构建产物条目数超过 %d 限制", maxZipEntries)
	}
	var found *zip.File
	for _, file := range reader.File {
		if file.FileInfo().IsDir() || !strings.HasSuffix(file.Name, ".zip") {
			continue
		}
		if !file.Mode().IsRegular() {
			return nil, errors.New("构建产物中的发布包不是普通文件")
		}
		if found != nil {
			return nil, errors.New("构建产物中有多个发布包，无法判断该用哪个")
		}
		found = file
	}
	if found == nil {
		return nil, errNoInnerArchive
	}
	return readZipEntry(found, MaxAssetBytes)
}

// errNoInnerArchive 表示外层产物里没有再套一层 zip，内容就平铺在外层。
var errNoInnerArchive = errors.New("构建产物中没有内层发布包")

func readZipEntry(file *zip.File, limit int64) ([]byte, error) {
	entry, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("读取 %s: %w", file.Name, err)
	}
	defer entry.Close()
	// 按 zip 声明的解压后大小预留容量。io.ReadAll 从 512 字节起反复翻倍，
	// 解一个 34MB 的二进制要重新分配并拷贝十几轮，白白冲高堆峰值。
	//
	// 声明值只当提示：它完全可能撒谎，因此既不按它预留超过上限的内存，
	// 也仍然用 LimitReader 兜住实际读入量。
	size := int64(file.UncompressedSize64)
	if size < 0 || size > limit {
		size = 0
	}
	buffer := bytes.NewBuffer(make([]byte, 0, size+1))
	if _, err := buffer.ReadFrom(io.LimitReader(entry, limit+1)); err != nil {
		return nil, fmt.Errorf("解压 %s: %w", file.Name, err)
	}
	if int64(buffer.Len()) > limit {
		return nil, fmt.Errorf("%s 解压后超过 %d 字节限制", file.Name, limit)
	}
	return buffer.Bytes(), nil
}
