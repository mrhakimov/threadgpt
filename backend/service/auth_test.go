package service

import "testing"

func TestAuthService_AuthorizeRejectsMalformedEncryptedToken(t *testing.T) {
	auth := NewAuthService()
	auth.SetEncryptionKey([]byte("0123456789abcdef0123456789abcdef"))
	auth.SetHashKey([]byte("fedcba9876543210fedcba9876543210"))

	auth.tokenStore["token"] = tokenEntry{
		EncryptedAPIKey: "YQ==",
		APIKeyHash:      "hash",
	}

	if _, err := auth.Authorize("token"); err == nil {
		t.Fatal("expected malformed ciphertext to be rejected")
	}
}
