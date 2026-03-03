package crypto

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"strings"

	"CrowdShardChain/internal/hash"
)

// Ed25519 公钥长度固定 32，签名长度固定 64。
const (
	PublicKeySize  = ed25519.PublicKeySize
	PrivateKeySize = ed25519.PrivateKeySize
	SignatureSize  = ed25519.SignatureSize
	AddressLen     = 20
)

// GenerateKey 生成一对 Ed25519 密钥。
func GenerateKey() (pub []byte, priv []byte, err error) {
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, nil, errors.New("生成Ed25519密钥失败")
	}
	return []byte(pubKey), []byte(privKey), nil
}

// Sign 对 msg 签名，返回 64 字节签名。
func Sign(privKey []byte, msg []byte) ([]byte, error) {
	if len(privKey) != PrivateKeySize {
		return nil, errors.New("签名失败：私钥长度不正确")
	}
	sig := ed25519.Sign(ed25519.PrivateKey(privKey), msg)
	if len(sig) != SignatureSize {
		return nil, errors.New("签名失败：签名长度异常")
	}
	return sig, nil
}

// Verify 验证签名。
func Verify(pubKey []byte, msg []byte, sig []byte) (bool, error) {
	if len(pubKey) != PublicKeySize {
		return false, errors.New("验签失败：公钥长度不正确")
	}
	if len(sig) != SignatureSize {
		return false, errors.New("验签失败：签名长度不正确")
	}
	ok := ed25519.Verify(ed25519.PublicKey(pubKey), msg, sig)
	return ok, nil
}

func AddressFromPublicKey(pubKey []byte) ([]byte, error) {
	if len(pubKey) != PublicKeySize {
		return nil, errors.New("生成地址失败：公钥长度不正确")
	}
	h := hash.Sum256(pubKey)
	var addr [20]byte
	copy(addr[:], h[:20])
	return addr[:], nil
}

func AddressHex(pubKey []byte) (string, error) {
	addr, err := AddressFromPublicKey(pubKey)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(addr[:]), nil
}

func ParseAddrHex(s string) ([]byte, error) {
	x := strings.ToLower(strings.TrimSpace(s))
	if len(x) != AddressLen*2 {
		return nil, errors.New("地址解析失败：长度必须为 40 位 hex")
	}
	b, err := hex.DecodeString(x)
	if err != nil {
		return nil, errors.New("地址解析失败：hex 非法")
	}
	var addr [AddressLen]byte
	copy(addr[:], b)
	return addr[:], nil
}
