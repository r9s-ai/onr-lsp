package lsp

import "testing"

func TestProviderNameFromURI(t *testing.T) {
	got := providerNameFromURI("file:///tmp/openai.conf")
	if got != "openai" {
		t.Fatalf("expected openai, got %q", got)
	}
	got = providerNameFromURI("/tmp/anthropic.conf")
	if got != "anthropic" {
		t.Fatalf("expected anthropic, got %q", got)
	}
}

func TestMaxHelper(t *testing.T) {
	if max(1, 2) != 2 {
		t.Fatalf("expected max(1,2)=2")
	}
	if max(5, 3) != 5 {
		t.Fatalf("expected max(5,3)=5")
	}
}
