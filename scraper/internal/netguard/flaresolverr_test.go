package netguard

import "testing"

func TestValidateFlareSolverrURL(t *testing.T) {
	for _, allowed := range []string{
		"",
		"http://flaresolverr:8191/v1",
		"http://127.0.0.1:8191/v1",
		"http://localhost:8191/v1",
	} {
		if err := ValidateFlareSolverrURL(allowed); err != nil {
			t.Fatalf("expected %q to be allowed: %v", allowed, err)
		}
	}

	for _, blocked := range []string{
		"https://example.com/v1",
		"http://user:pass@flaresolverr:8191/v1",
		"ftp://flaresolverr:8191/v1",
	} {
		if err := ValidateFlareSolverrURL(blocked); err == nil {
			t.Fatalf("expected %q to be blocked", blocked)
		}
	}
}
