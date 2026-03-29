package main

import "testing"

func TestShellSplitSimple(t *testing.T) {
	result := ShellSplit("--stamps-dir /tmp/s -v")
	if len(result) != 3 || result[0] != "--stamps-dir" || result[1] != "/tmp/s" || result[2] != "-v" {
		t.Fatalf("got %v", result)
	}
}

func TestShellSplitDoubleQuotes(t *testing.T) {
	result := ShellSplit(`--stamps-dir "/tmp/my stamps"`)
	if len(result) != 2 || result[1] != "/tmp/my stamps" {
		t.Fatalf("got %v", result)
	}
}

func TestShellSplitSingleQuotes(t *testing.T) {
	result := ShellSplit(`--stamps-dir '/tmp/my stamps'`)
	if len(result) != 2 || result[1] != "/tmp/my stamps" {
		t.Fatalf("got %v", result)
	}
}

func TestShellSplitBackslash(t *testing.T) {
	result := ShellSplit(`--stamps-dir /tmp/my\ stamps`)
	if len(result) != 2 || result[1] != "/tmp/my stamps" {
		t.Fatalf("got %v", result)
	}
}

func TestShellSplitEmpty(t *testing.T) {
	result := ShellSplit("")
	if len(result) != 0 {
		t.Fatalf("got %v", result)
	}
}

func TestShellSplitWhitespace(t *testing.T) {
	result := ShellSplit("  ")
	if len(result) != 0 {
		t.Fatalf("got %v", result)
	}
}
