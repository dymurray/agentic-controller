package git

import (
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

type Credentials struct {
	Username string
	Token    string
	RepoURL  string
	// PushURL, when set, is where the branch is pushed instead of RepoURL:
	// a fork the credential can write to when the application's repository
	// only grants it read access (HARNESS_PUSH_URL). The clone still comes
	// from RepoURL.
	PushURL string
	Branch  string
	// IdentityName is the Hub identity these credentials came from, empty
	// when none resolved and the run is anonymous. Carried so a failed
	// write-access pre-flight can name the credential that cannot push
	// instead of only saying that pushing failed (issue #247). Never the
	// secret itself — only the identity's display name.
	IdentityName string
}

// PushTarget is the repository pushes go to: PushURL when set, else
// RepoURL.
func (c *Credentials) PushTarget() string {
	if c.PushURL != "" {
		return c.PushURL
	}
	return c.RepoURL
}

// Auth returns nil (a truly nil interface, not a typed-nil *http.BasicAuth)
// when no identity was resolved: go-git treats any non-nil AuthMethod as an
// auth attempt, which non-HTTP transports (git://) reject with "invalid auth
// method" and the HTTP transport would deref.
func (c *Credentials) Auth() transport.AuthMethod {
	if c.Username == "" && c.Token == "" {
		return nil
	}
	return &http.BasicAuth{
		Username: c.Username,
		Password: c.Token,
	}
}
