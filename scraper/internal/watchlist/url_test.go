package watchlist

import "testing"

func TestValidateRemoteURL(t *testing.T) {
	if err := ValidateRemoteURL("https://raw.githubusercontent.com/aeswibon/manga-cdc/master/data/watchlist.yaml"); err != nil {
		t.Fatalf("expected allowlisted host to pass: %v", err)
	}
	if err := ValidateRemoteURL("http://raw.githubusercontent.com/a/b.yaml"); err == nil {
		t.Fatal("expected http to fail")
	}
	if err := ValidateRemoteURL("https://example.com/watchlist.yaml"); err == nil {
		t.Fatal("expected non-allowlisted host to fail")
	}
}
