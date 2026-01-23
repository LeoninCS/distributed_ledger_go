package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math/big"
)

// PublicKeyToHex 将公钥编码成稳定的 hex 字符串（长度恒为 128）。
func PublicKeyToHex(pub *ecdsa.PublicKey) (string, error) {
	b, err := PublicKeyToBytes(pub)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// HexToPublicKey 将地址 hex（128 个字符）还原为公钥。
func HexToPublicKey(addr string) (*ecdsa.PublicKey, error) {
	pubBytes, err := hex.DecodeString(addr)
	if err != nil {
		return nil, err
	}
	return BytesToPublishKey(pubBytes)
}

// 生成公私钥对
func GenerateKeyPair() (*ecdsa.PrivateKey, string, error) {
	pubCurve := elliptic.P256()
	privateKey, err := ecdsa.GenerateKey(pubCurve, rand.Reader)
	if err != nil {
		return nil, "", err
	}

	// 公钥编码为稳定的 16 进制作为用户地址（恒定 64 字节 => 128 hex 字符）
	address, err := PublicKeyToHex(&privateKey.PublicKey)
	if err != nil {
		return nil, "", err
	}

	return privateKey, address, nil
}

// 对数据签名
func Sign(privKey *ecdsa.PrivateKey, data []byte) ([]byte, error) {
	hash := sha256.Sum256(data)

	signature, err := ecdsa.SignASN1(rand.Reader, privKey, hash[:])
	if err != nil {
		return nil, err
	}

	return signature, nil
}

// 检验签名
func VerifyASN1Signature(pubKey *ecdsa.PublicKey, data []byte, signature []byte) bool {
	// 计算数据的哈希
	hash := sha256.Sum256(data)

	// 直接调用标准库提供的 ASN.1 校验函数
	return ecdsa.VerifyASN1(pubKey, hash[:], signature)
}

// PublicKeyToBytes 将 P-256 公钥编码为固定 64 字节：X(32) || Y(32)，左侧补 0。
// 这样 hex 后的“地址”长度恒定（128 个 hex 字符），可稳定反解析。
func PublicKeyToBytes(pub *ecdsa.PublicKey) ([]byte, error) {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return nil, errors.New("nil public key")
	}
	if pub.Curve == nil {
		pub.Curve = elliptic.P256()
	}
	if !pub.Curve.IsOnCurve(pub.X, pub.Y) {
		return nil, errors.New("invalid public key: point is not on curve")
	}

	b := make([]byte, 64)
	pub.X.FillBytes(b[:32])
	pub.Y.FillBytes(b[32:])
	return b, nil
}

func BytesToPublishKey(pubBytes []byte) (*ecdsa.PublicKey, error) {
	if len(pubBytes) != 64 {
		return nil, errors.New("invalid public key length: expected 64 bytes")
	}
	pubKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(pubBytes[:32]),
		Y:     new(big.Int).SetBytes(pubBytes[32:]),
	}
	if !pubKey.Curve.IsOnCurve(pubKey.X, pubKey.Y) {
		return nil, errors.New("invalid public key: point is not on the P256 curve")
	}
	return pubKey, nil
}
