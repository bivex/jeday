package service

import (
	"testing"
	"time"
)

func TestNewAuthServiceAppliesBatchOptions(t *testing.T) {
	svc := NewAuthService(nil, "12345678901234567890123456789012", Options{
		RegistrationBatchSize: 256,
		RegistrationBatchWait: 3 * time.Millisecond,
	}).(*authService)

	if svc.regBatchSize != 256 {
		t.Fatalf("regBatchSize = %d, want 256", svc.regBatchSize)
	}
	if svc.regBatchWait != 3*time.Millisecond {
		t.Fatalf("regBatchWait = %s, want 3ms", svc.regBatchWait)
	}
}

func TestNewAuthServiceUsesDefaultBatchOptions(t *testing.T) {
	svc := NewAuthService(nil, "12345678901234567890123456789012").(*authService)

	if svc.regBatchSize != 100 {
		t.Fatalf("regBatchSize = %d, want 100", svc.regBatchSize)
	}
	if svc.regBatchWait != 10*time.Millisecond {
		t.Fatalf("regBatchWait = %s, want 10ms", svc.regBatchWait)
	}
}
