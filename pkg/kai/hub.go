package kai

import (
	"crypto/tls"
	"fmt"
	"os"
	"strings"

	"github.com/konveyor/tackle2-hub/shared/api"
	"github.com/konveyor/tackle2-hub/shared/binding"
	"github.com/konveyor/tackle2-hub/shared/binding/auth"
	"github.com/spf13/cobra"
)

// defaultTokenLifespan is the lifespan, in hours, of the personal access token
// minted by 'hub login'. 720h is 30 days.
const defaultTokenLifespan = 720

func newHubCommand(cfg *kaiConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hub",
		Short: "Authenticate with Tackle Hub",
	}
	cmd.AddCommand(newHubLoginCommand(cfg))
	cmd.AddCommand(newHubLogoutCommand(cfg))
	return cmd
}

func newHubLoginCommand(cfg *kaiConfig) *cobra.Command {
	var (
		hubURL   string
		lifespan int
		insecure bool
	)
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to Tackle Hub and save a token for runs",
		Long: "Prompt for a Hub username and password, mint a personal access " +
			"token, and save it locally. Subsequent 'agent run' / 'workflow run' " +
			"invocations that target an application (--app) inject it as HUB_TOKEN.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHubLogin(hubURL, lifespan, insecure)
		},
	}
	cmd.Flags().StringVar(&hubURL, "hub-url", "", "Tackle Hub base URL (e.g. the OpenShift Route); required")
	cmd.Flags().IntVar(&lifespan, "lifespan", defaultTokenLifespan, "token lifespan in hours")
	cmd.Flags().BoolVar(&insecure, "insecure", false, "skip TLS certificate verification")
	_ = cmd.MarkFlagRequired("hub-url")
	return cmd
}

func runHubLogin(hubURL string, lifespan int, insecure bool) error {
	if err := requireTerminal(); err != nil {
		return err
	}
	hubURL = strings.TrimSpace(hubURL)
	if hubURL == "" {
		return fmt.Errorf("--hub-url is required")
	}

	var username, password string
	if err := runForm(inputField("Hub username", "admin", &username, requiredValidator("username"))); err != nil {
		return err
	}
	if err := runForm(passwordField("Hub password", &password, requiredValidator("password"))); err != nil {
		return err
	}

	// Build a Hub client, authenticate with Basic credentials, and mint a
	// durable personal access token to store. Basic auth carries the
	// credentials on every request; the token is what we persist, never the
	// password. SetRetry(1) makes a bad login fail fast instead of retrying.
	rc := binding.New(hubURL)
	rc.Client.SetRetry(1)
	if insecure {
		rc.Client.Transport().TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // opt-in via --insecure
	}
	rc.Client.Use(auth.NewBasic(strings.TrimSpace(username), password))

	pat := &api.PAT{Lifespan: lifespan, Description: "kubectl-kai"}
	if err := rc.Token.Create(pat); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	dir, err := hubConfigDir()
	if err != nil {
		return err
	}
	if err := saveHubToken(dir, hubCredentials{HubURL: hubURL, Token: pat.Token, Expiration: pat.Expiration}); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "logged in to %s; token saved to %s\n", hubURL, fmt.Sprintf("%s/%s", dir, hubTokenFile))
	return nil
}

func newHubLogoutCommand(cfg *kaiConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove the saved Tackle Hub token",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := hubConfigDir()
			if err != nil {
				return err
			}
			if err := deleteHubToken(dir); err != nil {
				return err
			}
			fmt.Fprintln(os.Stdout, "logged out; saved Hub token removed")
			return nil
		},
	}
}
