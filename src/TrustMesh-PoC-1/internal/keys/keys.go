package keys

import (
	"TrustMesh-PoC-1/internal/constants"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"unsafe"
)

//go:noinline
func Zeroize(b []byte) {
	if len(b) == 0 {
		return
	}

	ptr := unsafe.Pointer(&b[0])
	for i := 0; i < len(b); i++ {
		*(*byte)(unsafe.Add(ptr, i)) = 0
	}

	runtime.KeepAlive(b)
}

// seedPath seed 文件路径
func seedPath() string {
	path := filepath.Join(constants.ConfigDir, "seed.key")
	return path
}

// LoadOrCreateKey 返回公/私钥
func LoadOrCreateKey() (ed25519.PrivateKey, ed25519.PublicKey, error) {
	path := seedPath()

	// 创建目录
	if err := os.MkdirAll(filepath.Dir(path), 0o0700); err != nil {
		return nil, nil, fmt.Errorf("cannot create keys directory: %w", err)
	}

	// 如果文件不存在 → 创建新 seed
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		seed := make([]byte, ed25519.SeedSize)

		if _, err := rand.Read(seed); err != nil {
			return nil, nil, fmt.Errorf("cannot generate seed: %w", err)
		}

		// 写入 seed 文件（权限 0600）
		if err := os.WriteFile(path, seed, 0o0600); err != nil {
			Zeroize(seed)
			return nil, nil, fmt.Errorf("cannot write seed file: %w", err)
		}

		privateKey := ed25519.NewKeyFromSeed(seed)
		Zeroize(seed) // 清理 seed 内存

		return privateKey, privateKey.Public().(ed25519.PublicKey), nil
	}

	// 文件存在 → 尝试读取
	seed, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot read seed file: %w", err)
	}

	if len(seed) != ed25519.SeedSize {
		return nil, nil, fmt.Errorf("invalid seed length %d", len(seed))
	}

	// 尝试构建私钥
	privateKey := ed25519.NewKeyFromSeed(seed)
	Zeroize(seed)

	return privateKey, privateKey.Public().(ed25519.PublicKey), nil
}
