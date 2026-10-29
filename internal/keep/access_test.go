package keep

import "testing"

func TestEffectiveAccessEnvWins(t *testing.T) {
	getenv := func(k string) string {
		if k == "KEEP_ACCESS" {
			return "write"
		}
		return ""
	}
	mode, src := EffectiveAccess(getenv)
	if mode != "write" || src != "KEEP_ACCESS" {
		t.Fatalf("got %q %q", mode, src)
	}
}
