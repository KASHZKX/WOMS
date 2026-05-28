package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/d11nn/woms/internal/domain"
)

func TestCreateAndVerifyToken(t *testing.T) {
	token, err := CreateToken("secret", Claims{
		Subject: "user-1",
		Role:    domain.RoleScheduler,
		LineID:  "A",
	}, time.Hour)
	if err != nil {
		t.Fatalf("CreateToken returned error: %v", err)
	}

	claims, err := VerifyToken("secret", token)
	if err != nil {
		t.Fatalf("VerifyToken returned error: %v", err)
	}
	if claims.Subject != "user-1" || claims.Role != domain.RoleScheduler || claims.LineID != "A" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestVerifyTokenRejectsTampering(t *testing.T) {
	token, err := CreateToken("secret", Claims{Subject: "user-1", Role: domain.RoleSales}, time.Hour)
	if err != nil {
		t.Fatalf("CreateToken returned error: %v", err)
	}

	_, err = VerifyToken("other-secret", token)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestVerifyTokenRejectsExpiredToken(t *testing.T) {
	token, err := CreateToken("secret", Claims{Subject: "user-1", Role: domain.RoleSales}, -time.Hour)
	if err != nil {
		t.Fatalf("CreateToken returned error: %v", err)
	}

	_, err = VerifyToken("secret", token)
	if !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("expected ErrExpiredToken, got %v", err)
	}
}

func TestVerifyTokenRejectsInvalidRole(t *testing.T) {
	token, err := CreateToken("secret", Claims{Subject: "user-1", Role: domain.Role("owner")}, time.Hour)
	if err != nil {
		t.Fatalf("CreateToken returned error: %v", err)
	}

	_, err = VerifyToken("secret", token)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestVerifyTokenRejectsSchedulerWithoutLine(t *testing.T) {
	token, err := CreateToken("secret", Claims{Subject: "user-1", Role: domain.RoleScheduler}, time.Hour)
	if err != nil {
		t.Fatalf("CreateToken returned error: %v", err)
	}

	_, err = VerifyToken("secret", token)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestBearerToken(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		want    string
		wantErr bool
	}{
		{name: "empty", header: "", wantErr: true},
		{name: "malformed", header: "Bearer", wantErr: true},
		{name: "lowercase", header: "bearer token-1", wantErr: true},
		{name: "valid", header: "Bearer token-1", want: "token-1"},
		{name: "extra space trims token", header: "Bearer   token-1  ", want: "token-1"},
		{name: "non bearer", header: "Basic token-1", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BearerToken(tt.header)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidToken) {
					t.Fatalf("BearerToken error = %v, want ErrInvalidToken", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("BearerToken returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("BearerToken = %q, want %q", got, tt.want)
			}
		})
	}
}
