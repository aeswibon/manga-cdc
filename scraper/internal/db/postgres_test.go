package db

import (
	"context"
	"os"
	"testing"
)

func TestEncryptedCredentials(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping db test in short mode")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	database, err := New(ctx, dbURL, 5)
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}
	defer database.Close()

	key := "test-encryption-key-123"
	source := "test_source"
	payload := []byte(`{"token": "super-secret"}`)

	// Test Upsert
	err = database.UpsertCredential(ctx, source, payload, key)
	if err != nil {
		t.Fatalf("UpsertCredential failed: %v", err)
	}

	// Test Get
	retrieved, err := database.GetCredential(ctx, source, key)
	if err != nil {
		t.Fatalf("GetCredential failed: %v", err)
	}
	if string(retrieved) != string(payload) {
		t.Errorf("expected %s, got %s", payload, retrieved)
	}

	// Test Get with wrong key
	_, err = database.GetCredential(ctx, source, "wrong-key")
	if err == nil {
		t.Error("expected error when getting credential with wrong key, got nil")
	}
}
