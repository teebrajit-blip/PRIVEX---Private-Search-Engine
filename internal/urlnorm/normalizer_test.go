package urlnorm

import "testing"

func TestNormalize(t *testing.T) {
	got, err := Normalize("HTTPS://Example.COM:443/docs/page/?utm_source=test&b=2&a=1#part")
	if err != nil {
		t.Fatal(err)
	}
	want := "https://example.com/docs/page?a=1&b=2"
	if got != want {
		t.Fatalf("Normalize() = %q, want %q", got, want)
	}
}
