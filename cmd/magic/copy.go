package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"magic/internal/catalog"
	"magic/internal/core"
	"magic/internal/provider"
	"magic/internal/secrets"
)

func cmdCopy() *cobra.Command {
	var revealKey bool
	var protocol string
	var endpointID string
	c := &cobra.Command{
		Use:   "copy <provider>",
		Short: "Copy provider config from providers.yaml to stdout",
		Long: `Copy non-secret provider fields to stdout (clipboard-friendly).

Reads $MAGIC_HOME/providers.yaml. Each provider's key_env is a logical index
into local plaintext $MAGIC_HOME/secrets/providers.json (not a process env var).
.enc is the cloud state of that file after sync push — not a local store.

Key resolution (copy):
  1. read secrets/providers.json (plaintext, no passphrase)
  2. look up key_env (fallback: provider name)
  3. if found → print (default redact middle; --reveal-key for full value)
  4. if missing → placeholder <KEY_ENV> plus a one-line note

Process environment variables are never used for copy key resolution.

Endpoint selection:
  default → first endpoints[] entry (or synthesized base_url+protocol)
  --endpoint <id> → select by endpoint id
  --protocol <name> → first endpoint matching protocol (aliases ok)

Set keys with: magic secrets set <key_env> --value <secret>

Examples:
  magic copy deepseek
  magic copy shu --protocol anthropic-messages
  magic copy shu --endpoint openai-responses --reveal-key`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mh, err := resolveMagicHome()
			if err != nil {
				return err
			}
			name := args[0]
			// Effective catalog = builtin template overlaid by the user file, so a
			// builtin provider can be copied without adding it first.
			f, err := provider.LoadCatalog(mh)
			if err != nil {
				return err
			}
			p, ok := f.Providers[name]
			if !ok {
				return fmt.Errorf("provider %q not found in providers.yaml", name)
			}
			ep, err := catalog.ResolveCopyEndpoint(p, endpointID, protocol)
			if err != nil {
				return err
			}
			apiKey, err := resolveCopyAPIKey(core.NewPaths(mh).Secrets, name, p)
			if err != nil {
				return err
			}
			fmt.Print(catalog.CopyTextEndpoint(name, p, ep, apiKey, revealKey))
			return nil
		},
	}
	c.Flags().BoolVar(&revealKey, "reveal-key", false, "print full api key (default redacts middle)")
	c.Flags().StringVar(&protocol, "protocol", "", "select endpoint by protocol (e.g. anthropic-messages, openai-responses)")
	c.Flags().StringVar(&endpointID, "endpoint", "", "select endpoint by id from endpoints[]")
	return c
}

// resolveCopyAPIKey loads provider secrets and looks up key_env, then provider name.
func resolveCopyAPIKey(secretsDir, providerName string, p catalog.Provider) (string, error) {
	m, err := secrets.LoadProviderSecrets(secretsDir)
	if err != nil {
		return "", err
	}
	if key := strings.TrimSpace(p.KeyEnv); key != "" {
		if v, ok := m[key]; ok && v != "" {
			return v, nil
		}
	}
	// Fallback: provider name as index (e.g. "deepseek").
	if v, ok := m[providerName]; ok && v != "" {
		return v, nil
	}
	return "", nil
}
