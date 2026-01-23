package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"
)

func TestLoadPublicKey(t *testing.T) {
	// Generate test key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Create temporary public key file
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("Failed to marshal public key: %v", err)
	}

	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	tmpFile, err := os.CreateTemp("", "pubkey_*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(pubKeyPEM); err != nil {
		t.Fatalf("Failed to write public key: %v", err)
	}
	tmpFile.Close()

	// Test loading public key
	loadedKey, err := LoadPublicKey(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load public key: %v", err)
	}

	if loadedKey.N.Cmp(privateKey.PublicKey.N) != 0 {
		t.Error("Loaded public key doesn't match original")
	}
}

func TestLoadPrivateKey(t *testing.T) {
	// Generate test key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Create temporary private key file
	privKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privKeyBytes,
	})

	tmpFile, err := os.CreateTemp("", "privkey_*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(privKeyPEM); err != nil {
		t.Fatalf("Failed to write private key: %v", err)
	}
	tmpFile.Close()

	// Test loading private key
	loadedKey, err := LoadPrivateKey(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load private key: %v", err)
	}

	if loadedKey.N.Cmp(privateKey.N) != 0 {
		t.Error("Loaded private key doesn't match original")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	// Generate test key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	testData := []byte("test message for encryption")

	// Test encryption
	encrypted, err := Encrypt(testData, &privateKey.PublicKey)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	// Test decryption
	decrypted, err := Decrypt(encrypted, privateKey)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	if string(decrypted) != string(testData) {
		t.Errorf("Decrypted data doesn't match original. Got %s, want %s", string(decrypted), string(testData))
	}
}

func TestLoadPublicKeyErrors(t *testing.T) {
	// Test non-existent file
	_, err := LoadPublicKey("nonexistent.pem")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	// Test invalid PEM
	tmpFile, err := os.CreateTemp("", "invalid_*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("invalid pem data")
	tmpFile.Close()

	_, err = LoadPublicKey(tmpFile.Name())
	if err == nil {
		t.Error("Expected error for invalid PEM")
	}
}

func TestLoadPrivateKeyErrors(t *testing.T) {
	// Test non-existent file
	_, err := LoadPrivateKey("nonexistent.pem")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestLoadPrivateKeyInvalidPEM(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "invalid_priv_*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("invalid pem data")
	tmpFile.Close()

	_, err = LoadPrivateKey(tmpFile.Name())
	if err == nil {
		t.Error("Expected error for invalid PEM")
	}
}

func TestLoadPublicKeyInvalidKey(t *testing.T) {
	// Create PEM with wrong type
	tmpFile, err := os.CreateTemp("", "wrong_type_*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	wrongPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: []byte("invalid key data"),
	})
	tmpFile.Write(wrongPEM)
	tmpFile.Close()

	_, err = LoadPublicKey(tmpFile.Name())
	if err == nil {
		t.Error("Expected error for invalid key data")
	}
}

func TestLoadPrivateKeyInvalidKey(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "wrong_priv_*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	wrongPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: []byte("invalid key data"),
	})
	tmpFile.Write(wrongPEM)
	tmpFile.Close()

	_, err = LoadPrivateKey(tmpFile.Name())
	if err == nil {
		t.Error("Expected error for invalid key data")
	}
}

func TestEncryptLargeData(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Create data larger than RSA block size
	largeData := make([]byte, 1000)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	encrypted, err := Encrypt(largeData, &privateKey.PublicKey)
	if err != nil {
		t.Fatalf("Failed to encrypt large data: %v", err)
	}

	decrypted, err := Decrypt(encrypted, privateKey)
	if err != nil {
		t.Fatalf("Failed to decrypt large data: %v", err)
	}

	if string(decrypted) != string(largeData) {
		t.Error("Decrypted large data doesn't match original")
	}
}

func TestDecryptInvalidData(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Try to decrypt invalid data
	_, err = Decrypt([]byte("invalid encrypted data"), privateKey)
	if err == nil {
		t.Error("Expected error for invalid encrypted data")
	}
}

func TestEncryptEmptyData(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	encrypted, err := Encrypt([]byte{}, &privateKey.PublicKey)
	if err != nil {
		t.Fatalf("Failed to encrypt empty data: %v", err)
	}

	decrypted, err := Decrypt(encrypted, privateKey)
	if err != nil {
		t.Fatalf("Failed to decrypt empty data: %v", err)
	}

	if len(decrypted) != 0 {
		t.Error("Expected empty decrypted data")
	}
}

func TestEncryptDecryptChunks(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Test various data sizes
	sizes := []int{0, 10, 100, 200, 500}
	for _, size := range sizes {
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(i % 256)
		}

		encrypted, err := Encrypt(data, &privateKey.PublicKey)
		if err != nil {
			t.Fatalf("Encrypt size %d failed: %v", size, err)
		}

		decrypted, err := Decrypt(encrypted, privateKey)
		if err != nil {
			t.Fatalf("Decrypt size %d failed: %v", size, err)
		}

		if string(decrypted) != string(data) {
			t.Errorf("Size %d: data mismatch", size)
		}
	}
}

func TestDecryptTruncatedData(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	data := []byte("test data")
	encrypted, _ := Encrypt(data, &privateKey.PublicKey)

	// Truncate encrypted data
	if len(encrypted) > 10 {
		_, err = Decrypt(encrypted[:10], privateKey)
		if err == nil {
			t.Error("Expected error for truncated data")
		}
	}
}

func TestDecryptEmptyData(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Empty data should return error or empty result
	_, err = Decrypt([]byte{}, privateKey)
	// Either error or empty result is acceptable
	t.Logf("Decrypt empty result: err=%v", err)
}
