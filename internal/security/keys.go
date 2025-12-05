package security

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"sync"
)

// KeyManager manages RSA key pairs for JWT signing
type KeyManager struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	mu         sync.RWMutex
}

var (
	keyManager     *KeyManager
	keyManagerOnce sync.Once
)

// GetKeyManager returns the singleton key manager instance
func GetKeyManager() *KeyManager {
	keyManagerOnce.Do(func() {
		keyManager = &KeyManager{}
		if err := keyManager.initialize(); err != nil {
			panic(fmt.Sprintf("Failed to initialize key manager: %v", err))
		}
	})
	return keyManager
}

// initialize loads or generates RSA key pair
func (km *KeyManager) initialize() error {
	// Try to load existing keys
	privateKeyPath := os.Getenv("JWT_PRIVATE_KEY_PATH")
	publicKeyPath := os.Getenv("JWT_PUBLIC_KEY_PATH")

	if privateKeyPath != "" && publicKeyPath != "" {
		if err := km.loadKeys(privateKeyPath, publicKeyPath); err == nil {
			return nil
		}
	}

	// Generate new keys if loading failed
	return km.generateKeys()
}

// loadKeys loads RSA keys from PEM files
func (km *KeyManager) loadKeys(privateKeyPath, publicKeyPath string) error {
	// Load private key
	privateKeyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return err
	}

	block, _ := pem.Decode(privateKeyData)
	if block == nil {
		return fmt.Errorf("failed to decode private key PEM")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return err
	}

	// Load public key
	publicKeyData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return err
	}

	block, _ = pem.Decode(publicKeyData)
	if block == nil {
		return fmt.Errorf("failed to decode public key PEM")
	}

	publicKeyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return err
	}

	publicKey, ok := publicKeyInterface.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("not an RSA public key")
	}

	km.mu.Lock()
	km.privateKey = privateKey
	km.publicKey = publicKey
	km.mu.Unlock()

	return nil
}

// generateKeys generates a new RSA key pair
func (km *KeyManager) generateKeys() error {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	km.mu.Lock()
	km.privateKey = privateKey
	km.publicKey = &privateKey.PublicKey
	km.mu.Unlock()

	return nil
}

// GetPrivateKey returns the private key for signing
func (km *KeyManager) GetPrivateKey() *rsa.PrivateKey {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return km.privateKey
}

// GetPublicKey returns the public key for verification
func (km *KeyManager) GetPublicKey() *rsa.PublicKey {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return km.publicKey
}

// ExportPrivateKey exports the private key as PEM
func (km *KeyManager) ExportPrivateKey() ([]byte, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(km.privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	return privateKeyPEM, nil
}

// ExportPublicKey exports the public key as PEM
func (km *KeyManager) ExportPublicKey() ([]byte, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(km.publicKey)
	if err != nil {
		return nil, err
	}

	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	return publicKeyPEM, nil
}

// RotateKeys generates a new key pair (for key rotation)
func (km *KeyManager) RotateKeys() error {
	return km.generateKeys()
}
