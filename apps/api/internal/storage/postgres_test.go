package storage

import (
	"context"
	"testing"
)

func TestOpenPostgresRequiresDatabaseURL(t *testing.T) {
	_, err := OpenPostgres(context.Background(), "")
	if err == nil {
		t.Fatal("expected error")
	}
}
