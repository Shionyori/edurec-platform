package model_test

import (
	"testing"

	"github.com/Shionyori/edurec-platform/backend/internal/model"
)

func TestUserBeforeCreateInitializesEmptyInterests(t *testing.T) {
	user := &model.User{}
	if err := user.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate() error = %v", err)
	}
	if user.Interests != "[]" {
		t.Fatalf("Interests = %q, want []", user.Interests)
	}
}

func TestUserBeforeCreateKeepsExistingInterests(t *testing.T) {
	user := &model.User{Interests: `[1,3]`}
	if err := user.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate() error = %v", err)
	}
	if user.Interests != `[1,3]` {
		t.Fatalf("Interests = %q, want [1,3]", user.Interests)
	}
}
