package runner

import (
	"testing"
)

func TestProviderIsValid(t *testing.T) {
	tests := []struct {
		provider Provider
		valid    bool
	}{
		{ProviderClaude, true},
		{ProviderCopilot, true},
		{Provider(""), false},
		{Provider("openai"), false},
		{Provider("gemini"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			if got := tt.provider.IsValid(); got != tt.valid {
				t.Errorf("Provider(%q).IsValid() = %v, want %v", tt.provider, got, tt.valid)
			}
		})
	}
}

func TestProviderString(t *testing.T) {
	if got := ProviderClaude.String(); got != "claude" {
		t.Errorf("ProviderClaude.String() = %q, want %q", got, "claude")
	}
	if got := ProviderCopilot.String(); got != "copilot" {
		t.Errorf("ProviderCopilot.String() = %q, want %q", got, "copilot")
	}
}

func TestDefaultProvider(t *testing.T) {
	if DefaultProvider != ProviderClaude {
		t.Errorf("DefaultProvider = %q, want %q", DefaultProvider, ProviderClaude)
	}
}

func TestBinaryName(t *testing.T) {
	tests := []struct {
		provider Provider
		want     string
	}{
		{ProviderClaude, "claude"},
		{ProviderCopilot, "copilot"},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			if got := BinaryName(tt.provider); got != tt.want {
				t.Errorf("BinaryName(%q) = %q, want %q", tt.provider, got, tt.want)
			}
		})
	}
}

func TestNewRunnerUnknownProvider(t *testing.T) {
	_, err := NewRunner(Provider("unknown"), nil)
	if err == nil {
		t.Error("NewRunner with unknown provider should return error")
	}
}

func TestNewRunnerClaude(t *testing.T) {
	r, err := NewRunner(ProviderClaude, nil)
	if err != nil {
		t.Fatalf("NewRunner(claude) error: %v", err)
	}
	if r == nil {
		t.Fatal("NewRunner(claude) returned nil")
	}
}

func TestNewRunnerCopilot(t *testing.T) {
	r, err := NewRunner(ProviderCopilot, nil)
	if err != nil {
		t.Fatalf("NewRunner(copilot) error: %v", err)
	}
	if r == nil {
		t.Fatal("NewRunner(copilot) returned nil")
	}
}

func TestGenerateSessionID(t *testing.T) {
	id, err := GenerateSessionID()
	if err != nil {
		t.Fatalf("GenerateSessionID error: %v", err)
	}
	if id == "" {
		t.Error("GenerateSessionID returned empty string")
	}

	// Check uniqueness
	id2, err := GenerateSessionID()
	if err != nil {
		t.Fatalf("GenerateSessionID error: %v", err)
	}
	if id == id2 {
		t.Error("GenerateSessionID returned same ID twice")
	}
}
