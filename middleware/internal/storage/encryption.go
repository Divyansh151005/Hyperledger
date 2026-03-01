package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

// EncryptionService handles encryption and decryption of medical records
type EncryptionService struct {
	db         *Database
	keyStore   *KeyStore
	masterKey  []byte // Master key for encrypting other keys (should be from secure key management)
}

// EncryptionConfig holds encryption configuration
type EncryptionConfig struct {
	KeyStorePath string
	Algorithm    string
	KeySize      int
}

// NewEncryptionService creates a new encryption service
func NewEncryptionService(config EncryptionConfig) *EncryptionService {
	// In production, master key should come from a secure key management service
	// For now, we'll derive it from a fixed value (NOT SECURE FOR PRODUCTION)
	masterKey := pbkdf2.Key([]byte("master-key-change-in-production"), []byte("salt"), 4096, 32, sha256.New)

	return &EncryptionService{
		keyStore:  NewKeyStore(config.KeyStorePath),
		masterKey: masterKey,
	}
}

// SetDatabase sets the database connection for key storage
func (e *EncryptionService) SetDatabase(db *Database) {
	e.db = db
}

// Encrypt encrypts data using a key associated with authorizationRequestID
func (e *EncryptionService) Encrypt(data []byte, authorizationRequestID string) ([]byte, error) {
	// Get or generate key for this authorization request
	key, err := e.getOrGenerateKey(authorizationRequestID)
	if err != nil {
		return nil, fmt.Errorf("failed to get encryption key: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt
	ciphertext := gcm.Seal(nonce, nonce, data, nil)

	return ciphertext, nil
}

// Decrypt decrypts data using a key associated with authorizationRequestID
func (e *EncryptionService) Decrypt(encryptedData []byte, authorizationRequestID string) ([]byte, error) {
	// Get key for this authorization request
	key, err := e.getKey(authorizationRequestID)
	if err != nil {
		return nil, fmt.Errorf("failed to get decryption key: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce
	nonceSize := gcm.NonceSize()
	if len(encryptedData) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := encryptedData[:nonceSize], encryptedData[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// ComputeHash computes SHA-256 hash of data
func (e *EncryptionService) ComputeHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// getOrGenerateKey gets an existing key or generates a new one
func (e *EncryptionService) getOrGenerateKey(authorizationRequestID string) ([]byte, error) {
	// Try to get existing key from database
	if e.db != nil {
		key, err := e.getKeyFromDB(authorizationRequestID)
		if err == nil && key != nil {
			return key, nil
		}
	}

	// Generate new key
	key := make([]byte, 32) // 256 bits
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	// Store encrypted key in database
	if e.db != nil {
		if err := e.storeKeyInDB(authorizationRequestID, key); err != nil {
			return nil, fmt.Errorf("failed to store key: %w", err)
		}
	} else {
		// Fallback to file-based storage
		if err := e.keyStore.Store(authorizationRequestID, key, e.masterKey); err != nil {
			return nil, fmt.Errorf("failed to store key: %w", err)
		}
	}

	return key, nil
}

// getKey retrieves a key for authorizationRequestID
func (e *EncryptionService) getKey(authorizationRequestID string) ([]byte, error) {
	// Try database first
	if e.db != nil {
		key, err := e.getKeyFromDB(authorizationRequestID)
		if err == nil && key != nil {
			return key, nil
		}
	}

	// Fallback to file-based storage
	key, err := e.keyStore.Retrieve(authorizationRequestID, e.masterKey)
	if err != nil {
		return nil, fmt.Errorf("key not found for authorizationRequestID %s: %w", authorizationRequestID, err)
	}

	return key, nil
}

// getKeyFromDB retrieves a key from the database
func (e *EncryptionService) getKeyFromDB(authorizationRequestID string) ([]byte, error) {
	var encryptedKey []byte
	err := e.db.conn.QueryRow(
		"SELECT encrypted_key FROM encryption_keys WHERE authorization_request_id = $1",
		authorizationRequestID,
	).Scan(&encryptedKey)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("key not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query key: %w", err)
	}

	// Decrypt the key using master key
	key, err := e.decryptKey(encryptedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt key: %w", err)
	}

	return key, nil
}

// storeKeyInDB stores an encrypted key in the database
func (e *EncryptionService) storeKeyInDB(authorizationRequestID string, key []byte) error {
	// Encrypt the key using master key
	encryptedKey, err := e.encryptKey(key)
	if err != nil {
		return fmt.Errorf("failed to encrypt key: %w", err)
	}

	_, err = e.db.conn.Exec(
		"INSERT INTO encryption_keys (authorization_request_id, encrypted_key) VALUES ($1, $2) ON CONFLICT (authorization_request_id) DO NOTHING",
		authorizationRequestID,
		encryptedKey,
	)
	if err != nil {
		return fmt.Errorf("failed to store key: %w", err)
	}

	return nil
}

// encryptKey encrypts a key using the master key
func (e *EncryptionService) encryptKey(key []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.masterKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, key, nil)
	return ciphertext, nil
}

// decryptKey decrypts a key using the master key
func (e *EncryptionService) decryptKey(encryptedKey []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.masterKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(encryptedKey) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := encryptedKey[:nonceSize], encryptedKey[nonceSize:]
	key, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return key, nil
}

// KeyStore handles file-based key storage (fallback)
type KeyStore struct {
	path string
}

// NewKeyStore creates a new key store
func NewKeyStore(path string) *KeyStore {
	return &KeyStore{path: path}
}

// Store stores a key encrypted with master key
func (k *KeyStore) Store(id string, key []byte, masterKey []byte) error {
	// In production, implement proper file-based storage
	// For now, this is a placeholder
	return nil
}

// Retrieve retrieves a key and decrypts it
func (k *KeyStore) Retrieve(id string, masterKey []byte) ([]byte, error) {
	// In production, implement proper file-based retrieval
	// For now, this is a placeholder
	return nil, fmt.Errorf("key not found")
}
