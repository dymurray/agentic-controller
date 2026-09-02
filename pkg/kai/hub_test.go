package kai

import "testing"

func TestDeriveIssuerURL(t *testing.T) {
	tests := []struct {
		hubURL string
		want   string
	}{
		{"https://host/hub", "https://host/oidc"},
		{"https://host/hub/", "https://host/oidc"},
		{"https://host", "https://host/oidc"},
		{"https://host/", "https://host/oidc"},
		{"http://tackle-hub.konveyor-tackle.svc:8080", "http://tackle-hub.konveyor-tackle.svc:8080/oidc"},
	}
	for _, tt := range tests {
		if got := deriveIssuerURL(tt.hubURL); got != tt.want {
			t.Errorf("deriveIssuerURL(%q) = %q, want %q", tt.hubURL, got, tt.want)
		}
	}
}
