package security

import (
	"strings"
	"testing"
)

func TestNewEncryptionManager(t *testing.T) {
	tests := []struct {
		name      string
		masterKey string
		wantErr   bool
	}{
		{
			name:      "valid key",
			masterKey: "test-master-key-32-chars-minimum",
			wantErr:   false,
		},
		{
			name:      "short key",
			masterKey: "short",
			wantErr:   true,
		},
		{
			name:      "empty key",
			masterKey: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEncryptionManager(tt.masterKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEncryptionManager() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEncryptionManager_EncryptDecrypt(t *testing.T) {
	em, err := NewEncryptionManager("test-master-key-32-chars-minimum-secure")
	if err != nil {
		t.Fatalf("Failed to create encryption manager: %v", err)
	}

	tests := []struct {
		name      string
		plaintext string
		wantErr   bool
	}{
		{
			name:      "normal text",
			plaintext: "Hello, World!",
			wantErr:   false,
		},
		{
			name:      "empty string",
			plaintext: "",
			wantErr:   false,
		},
		{
			name:      "long text",
			plaintext: strings.Repeat("Long text for testing encryption with lots of content. ", 100),
			wantErr:   false,
		},
		{
			name:      "special characters",
			plaintext: "!@#$%^&*()_+-=[]{}|;:'\",.<>?/~`",
			wantErr:   false,
		},
		{
			name:      "unicode",
			plaintext: "Hello 世界 🌍 مرحبا",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test encryption
			encrypted, err := em.Encrypt(tt.plaintext)
			if (err != nil) != tt.wantErr {
				t.Errorf("Encrypt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return // Skip decryption test if encryption failed
			}

			// Test that encrypted text is different from plaintext (unless empty)
			if tt.plaintext != "" && encrypted == tt.plaintext {
				t.Error("Encrypted text should be different from plaintext")
			}

			// Test decryption
			decrypted, err := em.Decrypt(encrypted)
			if err != nil {
				t.Errorf("Decrypt() error = %v", err)
				return
			}

			// Verify decrypted text matches original
			if decrypted != tt.plaintext {
				t.Errorf("Decrypted text = %v, want %v", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncryptionManager_DecryptInvalidData(t *testing.T) {
	em, err := NewEncryptionManager("test-master-key-32-chars-minimum-secure")
	if err != nil {
		t.Fatalf("Failed to create encryption manager: %v", err)
	}

	tests := []struct {
		name       string
		ciphertext string
		wantErr    bool
	}{
		{
			name:       "invalid base64",
			ciphertext: "invalid-base64-data!@#",
			wantErr:    true,
		},
		{
			name:       "short ciphertext",
			ciphertext: "c2hvcnQ=", // "short" in base64
			wantErr:    true,
		},
		{
			name:       "wrong key data",
			ciphertext: "dGVzdGluZyBpbnZhbGlkIGRhdGE=", // "testing invalid data" in base64
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := em.Decrypt(tt.ciphertext)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decrypt() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMaskSensitiveData(t *testing.T) {
	tests := []struct {
		name       string
		data       string
		showLength int
		expected   string
	}{
		{
			name:       "normal case",
			data:       "abcdefghijklmnop",
			showLength: 4,
			expected:   "abcd********mnop",
		},
		{
			name:       "short string",
			data:       "short",
			showLength: 4,
			expected:   "***",
		},
		{
			name:       "empty string",
			data:       "",
			showLength: 4,
			expected:   "",
		},
		{
			name:       "zero show length",
			data:       "abcdefghijklmnop",
			showLength: 0,
			expected:   "abcd********mnop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskSensitiveData(tt.data, tt.showLength)
			if result != tt.expected {
				t.Errorf("MaskSensitiveData() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGenerateSecureKey(t *testing.T) {
	tests := []struct {
		name    string
		length  int
		wantErr bool
	}{
		{
			name:    "normal length",
			length:  32,
			wantErr: false,
		},
		{
			name:    "zero length",
			length:  0,
			wantErr: true,
		},
		{
			name:    "negative length",
			length:  -1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := GenerateSecureKey(tt.length)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateSecureKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(key) == 0 {
				t.Error("GenerateSecureKey() should return non-empty key")
			}

			// Test that multiple calls return different keys
			if !tt.wantErr {
				key2, _ := GenerateSecureKey(tt.length)
				if key == key2 {
					t.Error("GenerateSecureKey() should return different keys on multiple calls")
				}
			}
		})
	}
}

func TestValidateKeyStrength(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{
			name:    "strong key",
			key:     "StrongPassword123WithMixedCaseAndNumbers",
			wantErr: false,
		},
		{
			name:    "too short",
			key:     "Short1A",
			wantErr: true,
		},
		{
			name:    "no uppercase",
			key:     "lowercasepassword123withnouppercase",
			wantErr: true,
		},
		{
			name:    "no lowercase",
			key:     "UPPERCASEPASSWORD123WITHNOLOWERCASE",
			wantErr: true,
		},
		{
			name:    "no digits",
			key:     "PasswordWithoutAnyDigitsButHasCase",
			wantErr: true,
		},
		{
			name:    "minimum valid",
			key:     "MinimumValidPassword123WithCase",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKeyStrength(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateKeyStrength() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEncryptionConsistency(t *testing.T) {
	masterKey := "test-master-key-32-chars-minimum-secure"
	
	// Create two encryption managers with the same key
	em1, err := NewEncryptionManager(masterKey)
	if err != nil {
		t.Fatalf("Failed to create first encryption manager: %v", err)
	}

	em2, err := NewEncryptionManager(masterKey)
	if err != nil {
		t.Fatalf("Failed to create second encryption manager: %v", err)
	}

	plaintext := "Test data for consistency check"

	// Encrypt with first manager
	encrypted1, err := em1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("First encryption failed: %v", err)
	}

	// Decrypt with second manager (should work due to same key)
	decrypted, err := em2.Decrypt(encrypted1)
	if err != nil {
		t.Fatalf("Cross-decryption failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Cross-decryption result = %v, want %v", decrypted, plaintext)
	}
}

func BenchmarkEncrypt(b *testing.B) {
	em, _ := NewEncryptionManager("test-master-key-32-chars-minimum-secure")
	plaintext := "This is a test string for benchmarking encryption performance"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = em.Encrypt(plaintext)
	}
}

func BenchmarkDecrypt(b *testing.B) {
	em, _ := NewEncryptionManager("test-master-key-32-chars-minimum-secure")
	plaintext := "This is a test string for benchmarking decryption performance"
	encrypted, _ := em.Encrypt(plaintext)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = em.Decrypt(encrypted)
	}
}