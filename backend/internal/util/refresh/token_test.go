package refresh_test

import (
	"testing"

	"github.com/Shionyori/edurec-platform/backend/internal/util/refresh"
)

func TestNewTokenGeneratesUniqueTokens(t *testing.T) {
	first, err := refresh.NewToken()
	if err != nil {
		t.Fatalf("NewToken() error = %v", err)
	}
	second, err := refresh.NewToken()
	if err != nil {
		t.Fatalf("NewToken() error = %v", err)
	}

	if first == "" || second == "" {
		t.Fatalf("NewToken() returned empty token: first=%q second=%q", first, second)
	}
	if first == second {
		t.Fatalf("NewToken() returned duplicate token %q", first)
	}
}

func TestKeyIsDeterministic(t *testing.T) {
	token := "refresh-token-value"
	if refresh.Key(token) != refresh.Key(token) {
		t.Fatal("Key() is not deterministic")
	}
	if refresh.Key(token) == refresh.Key("other-token") {
		t.Fatal("Key() should differ for different tokens")
	}
}
