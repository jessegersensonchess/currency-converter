package main

import (
	"os"
	"testing"
)

func TestDefaultPort_WithFlag(t *testing.T) {
	// If the user passes -p, that should win immediately.
	got := defaultPort("4000")
	want := "4000"
	if got != want {
		t.Errorf("defaultPort(\"4000\") = %q; want %q", got, want)
	}
}

func TestDefaultPort_WithEnv(t *testing.T) {
	// Clear any existing env, then set our test value.
	os.Unsetenv("CURRENCY_CONVERTER_PORT")
	defer os.Unsetenv("CURRENCY_CONVERTER_PORT")

	if err := os.Setenv("CURRENCY_CONVERTER_PORT", "3000"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}

	got := defaultPort("")
	want := "3000"
	if got != want {
		t.Errorf("defaultPort(\"\") with CURRENCY_CONVERTER_PORT=3000 = %q; want %q", got, want)
	}
}

func TestDefaultPort_Default(t *testing.T) {
	// No flag, no env ⇒ use the hard‐coded default.
	os.Unsetenv("CURRENCY_CONVERTER_PORT")
	got := defaultPort("")
	want := "18880"
	if got != want {
		t.Errorf("defaultPort(\"\") with no env = %q; want %q", got, want)
	}
}
