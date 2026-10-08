package encryption

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

const (
	privateKeyEnv     = "DCSY_RSA_PRIVATE_KEY"
	privateKeyFileEnv = "DCSY_RSA_PRIVATE_KEY_FILE"
	publicKeyEnv      = "DCSY_RSA_PUBLIC_KEY"
	publicKeyFileEnv  = "DCSY_RSA_PUBLIC_KEY_FILE"
)

// GenerateRSAKey 生成 PKCS#1 私钥和 PKIX 公钥。密钥只返回给调用方，不写日志。
func GenerateRSAKey() (privateByteBase64, publicByteBase64 string, err error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}
	privateDER := x509.MarshalPKCS1PrivateKey(privateKey)
	publicDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", err
	}
	return base64.StdEncoding.EncodeToString(privateDER), base64.StdEncoding.EncodeToString(publicDER), nil
}

// GenerateRSAPKCS8Key 生成 PKCS#8 私钥和 PKIX 公钥，供部署时创建新密钥对。
func GenerateRSAPKCS8Key() (privateByteBase64, publicByteBase64 string, err error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}
	privateDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return "", "", err
	}
	publicDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", err
	}
	return base64.StdEncoding.EncodeToString(privateDER), base64.StdEncoding.EncodeToString(publicDER), nil
}

// PKCS8Encrypt 使用指定公钥加密。兼容旧调用签名，失败时返回空字符串。
func PKCS8Encrypt(pubKey, data string) string {
	result, err := encryptWithPublicKey(pubKey, data)
	if err != nil {
		return ""
	}
	return result
}

// PKCS8Decrypt 使用指定私钥解密。兼容旧调用签名，失败时返回空字符串。
func PKCS8Decrypt(prvKey, text string) string {
	result, err := decryptWithPrivateKey(prvKey, text)
	if err != nil {
		return ""
	}
	return result
}

// DefaultPKCS8Encrypt 从环境变量或本地密钥文件读取公钥后加密。
func DefaultPKCS8Encrypt(data string) (string, error) {
	publicKey, err := loadKey(publicKeyEnv, publicKeyFileEnv)
	if err != nil {
		return "", err
	}
	return encryptWithPublicKey(publicKey, data)
}

// DefaultPKCS8Decrypt 从环境变量或本地密钥文件读取私钥后解密。
// 源码和 YAML 配置中不再保存固定私钥。
func DefaultPKCS8Decrypt(text string) (string, error) {
	privateKey, err := loadKey(privateKeyEnv, privateKeyFileEnv)
	if err != nil {
		return "", err
	}
	return decryptWithPrivateKey(privateKey, text)
}

func loadKey(valueEnv, fileEnv string) (string, error) {
	// 优先读取当前服务配置文件，满足本地部署直接使用 YAML 配置密钥的场景。
	configKey := ""
	switch valueEnv {
	case privateKeyEnv:
		configKey = "rsa.privateKey"
	case publicKeyEnv:
		configKey = "rsa.publicKey"
	}
	if configKey != "" {
		if value := strings.TrimSpace(viper.GetString(configKey)); value != "" {
			return value, nil
		}
	}

	// 保留环境变量和密钥文件方式，兼容已有部署环境。
	if value := strings.TrimSpace(os.Getenv(valueEnv)); value != "" {
		return value, nil
	}
	if path := strings.TrimSpace(os.Getenv(fileEnv)); path != "" {
		value, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("读取 RSA 密钥文件失败: %w", err)
		}
		if strings.TrimSpace(string(value)) == "" {
			return "", errors.New("RSA 密钥文件为空")
		}
		return string(value), nil
	}
	return "", fmt.Errorf("未配置 %s、%s 或配置项 %s", valueEnv, fileEnv, configKey)
}

func encryptWithPublicKey(keyText, data string) (string, error) {
	keyDER, err := decodeKeyMaterial(keyText)
	if err != nil {
		return "", fmt.Errorf("解析 RSA 公钥失败: %w", err)
	}
	parsed, err := x509.ParsePKIXPublicKey(keyDER)
	if err != nil {
		return "", fmt.Errorf("解析 RSA 公钥失败: %w", err)
	}
	publicKey, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return "", errors.New("配置的公钥不是 RSA 公钥")
	}
	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, []byte(data))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func decryptWithPrivateKey(keyText, text string) (string, error) {
	keyDER, err := decodeKeyMaterial(keyText)
	if err != nil {
		return "", fmt.Errorf("解析 RSA 私钥失败: %w", err)
	}
	parsed, err := x509.ParsePKCS8PrivateKey(keyDER)
	if err != nil {
		// 兼容 PKCS#1 文件，便于部署侧平滑轮换。
		if privateKey, pkcs1Err := x509.ParsePKCS1PrivateKey(keyDER); pkcs1Err == nil {
			return decrypt(privateKey, text)
		}
		return "", fmt.Errorf("解析 RSA 私钥失败: %w", err)
	}
	privateKey, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return "", errors.New("配置的私钥不是 RSA 私钥")
	}
	return decrypt(privateKey, text)
}

func decrypt(privateKey *rsa.PrivateKey, text string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(strings.TrimSpace(text))
	if err != nil {
		return "", fmt.Errorf("密文不是有效的 Base64: %w", err)
	}
	plaintext, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, ciphertext)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func decodeKeyMaterial(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if block, _ := pem.Decode([]byte(value)); block != nil {
		return block.Bytes, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}
