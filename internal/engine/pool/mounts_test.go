package pool

import (
	"errors"
	"os/user"
	"testing"
)

func TestGetUserString_Success(t *testing.T) {
	// Mock user.Current
	original := userCurrent
	defer func() { userCurrent = original }()

	userCurrent = func() (*user.User, error) {
		return &user.User{
			Uid: "1001",
			Gid: "1002",
		}, nil
	}

	got, err := getUserString()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "1001:1002"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestGetUserString_Error(t *testing.T) {
	// Mock user.Current failure
	original := userCurrent
	defer func() { userCurrent = original }()

	userCurrent = func() (*user.User, error) {
		return nil, errors.New("mock error")
	}

	_, err := getUserString()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
