package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"os"
)

func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}

	return rsaPub, nil
}

func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return priv, nil
}

// Encrypt использует гибридное шифрование: AES для данных, RSA для ключа AES
func Encrypt(data []byte, publicKey *rsa.PublicKey) ([]byte, error) {
	// Генерируем случайный AES ключ (256 бит)
	aesKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, aesKey); err != nil {
		return nil, fmt.Errorf("failed to generate AES key: %w", err)
	}

	// Шифруем AES ключ с помощью RSA
	encryptedKey, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt AES key: %w", err)
	}

	// Шифруем данные с помощью AES-GCM
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	encryptedData := gcm.Seal(nonce, nonce, data, nil)

	// Формат: [2 байта длины ключа][зашифрованный ключ][зашифрованные данные]
	keyLen := len(encryptedKey)
	result := make([]byte, 2+keyLen+len(encryptedData))
	result[0] = byte(keyLen >> 8)
	result[1] = byte(keyLen)
	copy(result[2:], encryptedKey)
	copy(result[2+keyLen:], encryptedData)

	return result, nil
}

// Decrypt расшифровывает данные, зашифрованные гибридным методом
func Decrypt(data []byte, privateKey *rsa.PrivateKey) ([]byte, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("data too short")
	}

	// Читаем длину зашифрованного ключа
	keyLen := int(data[0])<<8 | int(data[1])
	if len(data) < 2+keyLen {
		return nil, fmt.Errorf("invalid encrypted data format")
	}

	// Расшифровываем AES ключ
	encryptedKey := data[2 : 2+keyLen]
	aesKey, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, encryptedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt AES key: %w", err)
	}

	// Расшифровываем данные
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	encryptedData := data[2+keyLen:]
	if len(encryptedData) < gcm.NonceSize() {
		return nil, fmt.Errorf("encrypted data too short")
	}

	nonce := encryptedData[:gcm.NonceSize()]
	ciphertext := encryptedData[gcm.NonceSize():]

	return gcm.Open(nil, nonce, ciphertext, nil)
}
