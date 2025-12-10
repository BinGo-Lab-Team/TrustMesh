package p2p

import (
	"TrustMesh-PoC-1/internal/keys"
	"TrustMesh-PoC-1/internal/logger"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"time"

	"github.com/zeebo/blake3"
)

const maxDiff uint64 = 3600

// timeDiff 检验于当前时间的时间差
func timeDiff(remote uint64) bool {
	local := uint64(time.Now().UnixMilli())
	var diff uint64
	if remote > local {
		diff = remote - local
	} else {
		diff = local - remote
	}

	if diff < maxDiff {
		return true
	} else {
		return false
	}
}

// newHandshakeHello 生成Hello消息
func newHandshakeHello() ([76]byte, HandshakeMetadata, bool) {
	// 获取密钥
	_, publicKey, err := keys.LoadOrCreateKey()
	if err != nil {
		logger.Debug("Failed to load or create key: %v", err)
		return [76]byte{}, HandshakeMetadata{}, false
	}

	// 定义 Hello 结构体
	var payloadHello HandshakeMetadata

	// 写入公钥
	copy(payloadHello.PK[:], publicKey)
	// 写入随机数
	if _, err := rand.Read(payloadHello.Nonce[:]); err != nil {
		logger.Debug("Get nonce failed: %v", err)
		return [76]byte{}, HandshakeMetadata{}, false
	}
	// 写入日期
	payloadHello.Time = uint64(time.Now().UnixMilli())

	// 创建 [8]byte 格式的日期
	var HelloTime [8]byte
	binary.BigEndian.PutUint64(HelloTime[:], payloadHello.Time)
	// 创建 [4]byte 格式的 header
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], MsgHandshakeHello)

	var message [4 + 32 + 32 + 8]byte
	copy(message[0:4], header[:])
	copy(message[4:36], payloadHello.PK[:])
	copy(message[36:68], payloadHello.Nonce[:])
	copy(message[68:76], HelloTime[:])

	return message, payloadHello, true
}

// processingHandshakeHello 处理Hello消息
// 返回值：发送信息，对方握手信息，本地握手信息，是否处理成功
func processingHandshakeHello(body [72]byte) ([140]byte, HandshakeMetadata, HandshakeMetadata, bool) {
	// 获取密钥
	privateKey, publicKey, err := keys.LoadOrCreateKey()
	if err != nil {
		logger.Debug("Failed to load or create key: %v", err)
		return [140]byte{}, HandshakeMetadata{}, HandshakeMetadata{}, false
	}

	// 定义 Hello 结构体
	var payloadHello HandshakeMetadata

	// 拆分公钥
	copy(payloadHello.PK[:], body[0:32])
	// 拆分随机数
	copy(payloadHello.Nonce[:], body[32:64])
	// 拆分日期
	payloadHello.Time = binary.BigEndian.Uint64(body[64:72])
	// 判断日期误差
	if !timeDiff(payloadHello.Time) {
		return [140]byte{}, HandshakeMetadata{}, HandshakeMetadata{}, false
	}

	// 定义 Response 结构体
	var payloadResponse HandshakeMetadata

	// 写入公钥
	copy(payloadResponse.PK[:], publicKey)
	// 写入随机数
	if _, err := rand.Read(payloadResponse.Nonce[:]); err != nil {
		logger.Error("Failed to generate nonce: %v", err)
		return [140]byte{}, HandshakeMetadata{}, HandshakeMetadata{}, false
	}
	// 写入日期
	payloadResponse.Time = uint64(time.Now().UnixMilli())

	// 创建 [8]byte 格式的日期
	var responseTime [8]byte
	binary.BigEndian.PutUint64(responseTime[:], payloadResponse.Time)
	// 创建 [4]byte 格式的 Tag
	var TMHSV1DomainTag [4]byte
	binary.BigEndian.PutUint32(TMHSV1DomainTag[:], TMHSV1Domain)
	// 创建 [4]byte 格式的 header
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], MsgHandshakeResponse)

	// 组合挑战内容
	var challenge [4 + 72 + 32 + 32 + 8]byte
	copy(challenge[0:4], TMHSV1DomainTag[:])
	copy(challenge[4:76], body[:])
	copy(challenge[76:108], payloadResponse.PK[:])
	copy(challenge[108:140], payloadResponse.Nonce[:])
	copy(challenge[140:148], responseTime[:])

	// 组合发送消息
	challengeHash := blake3.Sum256(challenge[:])
	challengeSign := ed25519.Sign(privateKey, challengeHash[:])

	var message [4 + 64 + 32 + 32 + 8]byte
	copy(message[0:4], header[:])
	copy(message[4:68], challengeSign)
	copy(message[68:100], payloadResponse.PK[:])
	copy(message[100:132], payloadResponse.Nonce[:])
	copy(message[132:140], responseTime[:])

	keys.Zeroize(privateKey)

	return message, payloadHello, payloadResponse, true
}

// processingHandshakeResponse 处理Response消息
func processingHandshakeResponse(body [136]byte, payloadHello HandshakeMetadata) ([68]byte, HandshakeMetadata, bool) {
	// 获取密钥
	privateKey, _, err := keys.LoadOrCreateKey()
	if err != nil {
		logger.Debug("Failed to load or create key: %v", err)
		return [68]byte{}, HandshakeMetadata{}, false
	}

	// 定义 Response 结构体
	var payloadResponse HandshakeMetadata
	var challengeSignResponse [64]byte

	// 拆分签名
	copy(challengeSignResponse[:], body[0:64])
	// 拆分公钥
	copy(payloadResponse.PK[:], body[64:96])
	// 拆分随机数
	copy(payloadResponse.Nonce[:], body[96:128])
	// 拆分日期
	payloadResponse.Time = binary.BigEndian.Uint64(body[128:136])
	// 判断日期误差
	if !timeDiff(payloadResponse.Time) {
		logger.Debug("Received time is too far away")
		return [68]byte{}, HandshakeMetadata{}, false
	}

	// 创建 [8]byte 格式的日期
	var responseTime [8]byte
	binary.BigEndian.PutUint64(responseTime[:], payloadResponse.Time)
	// 创建 [8]byte 格式的日期
	var helloTime [8]byte
	binary.BigEndian.PutUint64(helloTime[:], payloadHello.Time)
	// 创建 [4]byte 格式的 Tag
	var TMHSV1DomainTag [4]byte
	binary.BigEndian.PutUint32(TMHSV1DomainTag[:], TMHSV1Domain)
	// 创建 [4]byte 格式的 header
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], MsgHandshakeConfirm)

	// 组合挑战内容
	var challenge [4 + 32 + 32 + 8 + 32 + 32 + 8]byte
	copy(challenge[0:4], TMHSV1DomainTag[:])
	copy(challenge[4:36], payloadHello.PK[:])
	copy(challenge[36:68], payloadHello.Nonce[:])
	copy(challenge[68:76], helloTime[:])
	copy(challenge[76:108], payloadResponse.PK[:])
	copy(challenge[108:140], payloadResponse.Nonce[:])
	copy(challenge[140:148], responseTime[:])
	challengeHash := blake3.Sum256(challenge[:])

	if !ed25519.Verify(payloadResponse.PK[:], challengeHash[:], challengeSignResponse[:]) {
		return [68]byte{}, HandshakeMetadata{}, false
	}

	challengeSign := ed25519.Sign(privateKey, challengeHash[:])
	var message [4 + 64]byte
	copy(message[0:4], header[:])
	copy(message[4:68], challengeSign)

	keys.Zeroize(privateKey)

	return message, payloadResponse, true
}

// processingHandshakeConfirm 确认返回
func processingHandshakeConfirm(body [64]byte, payloadHello HandshakeMetadata, payloadResponse HandshakeMetadata) bool {

	// 创建 [8]byte 格式的日期
	var helloTime [8]byte
	binary.BigEndian.PutUint64(helloTime[:], payloadHello.Time)
	// 创建 [8]byte 格式的日期
	var responseTime [8]byte
	binary.BigEndian.PutUint64(responseTime[:], payloadResponse.Time)
	// 创建 [4]byte 格式的 Tag
	var TMHSV1DomainTag [4]byte
	binary.BigEndian.PutUint32(TMHSV1DomainTag[:], TMHSV1Domain)

	// 组合挑战内容
	var challenge [4 + 72 + 32 + 32 + 8]byte
	copy(challenge[0:4], TMHSV1DomainTag[:])
	copy(challenge[4:36], payloadHello.PK[:])
	copy(challenge[36:68], payloadHello.Nonce[:])
	copy(challenge[68:76], helloTime[:])
	copy(challenge[76:108], payloadResponse.PK[:])
	copy(challenge[108:140], payloadResponse.Nonce[:])
	copy(challenge[140:148], responseTime[:])

	challengeHash := blake3.Sum256(challenge[:])

	return ed25519.Verify(payloadHello.PK[:], challengeHash[:], body[:])
}
