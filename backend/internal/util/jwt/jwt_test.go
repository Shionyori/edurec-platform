package jwt_test

import (
	"testing"
	"time"

	jwtutil "github.com/Shionyori/edurec-platform/backend/internal/util/jwt"
)

func TestGenerateAndParseAccessToken(t *testing.T) {
	manager := jwtutil.NewManager("test-secret")

	token, err := manager.GenerateAccessToken(42, 15*time.Minute)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	userID, err := manager.ParseAccessToken(token)
	if err != nil {
		t.Fatalf("ParseAccessToken() error = %v", err)
	}
	if userID != 42 {
		t.Fatalf("ParseAccessToken() userID = %d, want 42", userID)
	}
}

func TestParseExpiredAccessToken(t *testing.T) {
	manager := jwtutil.NewManager("test-secret")

	token, err := manager.GenerateAccessToken(1, -time.Minute)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	_, err = manager.ParseAccessToken(token)
	if err == nil {
		t.Fatal("ParseAccessToken() expected error for expired token")
	}
}

func TestParseRejectsTokenSignedWithOtherSecret(t *testing.T) {
	manager := jwtutil.NewManager("test-secret")
	other := jwtutil.NewManager("other-secret")

	token, err := other.GenerateAccessToken(1, time.Minute)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	_, err = manager.ParseAccessToken(token)
	if err == nil {
		t.Fatal("ParseAccessToken() expected error for wrong secret")
	}
}
