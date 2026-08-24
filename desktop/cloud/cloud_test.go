package cloud

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyDotEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := "# comment\nSUPABASE_URL=https://example.supabase.co\n\"QUOTED_KEY\" = \"abc def\"\nEMPTY=\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	if !applyDotEnv(path) {
		t.Fatal("expected .env to be applied")
	}
	if got := os.Getenv("SUPABASE_URL"); got != "https://example.supabase.co" {
		t.Fatalf("SUPABASE_URL = %q", got)
	}
	if got := os.Getenv("QUOTED_KEY"); got != "abc def" {
		t.Fatalf("QUOTED_KEY = %q", got)
	}

	t.Setenv("SUPABASE_URL", "https://preset.supabase.co")
	applyDotEnv(path)
	if got := os.Getenv("SUPABASE_URL"); got != "https://preset.supabase.co" {
		t.Fatalf("real env must win, got %q", got)
	}
}
