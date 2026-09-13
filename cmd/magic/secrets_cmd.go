package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"magic/internal/core"
	"magic/internal/secrets"
)

func cmdSecrets() *cobra.Command {
	c := &cobra.Command{
		Use:   "secrets",
		Short: "Manage provider API keys (local plaintext store)",
		Long: `Provider API keys live in local plaintext
$MAGIC_HOME/secrets/providers.json (0600), indexed by key_env from
providers.yaml. No passphrase. Local is never encrypted.

.enc is the cloud state of secret-bearing files after magic sync push.
Daily commands (apply / copy / secrets / provider) only read the
plaintext file.`,
	}
	c.AddCommand(cmdSecretsSet())
	c.AddCommand(cmdSecretsList())
	return c
}

func cmdSecretsSet() *cobra.Command {
	var valueFlag string
	c := &cobra.Command{
		Use:   "set <key_env>",
		Short: "Set a provider API key in secrets/providers.json",
		Long: `Merge <key_env> → value into local plaintext secrets/providers.json.
No passphrase.

Prefer --value (or stdin) so shells do not store the secret in history:

  magic secrets set DEEPSEEK_API_KEY --value 'sk-…'
  echo 'sk-…' | magic secrets set DEEPSEEK_API_KEY
  magic secrets set DEEPSEEK_API_KEY   # reads value from stdin`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			keyEnv := strings.TrimSpace(args[0])
			if keyEnv == "" {
				return fmt.Errorf("key_env must be non-empty")
			}
			val := valueFlag
			if val == "" {
				// Read from stdin when --value omitted.
				data, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("read value from stdin: %w", err)
				}
				val = strings.TrimRight(string(data), "\r\n")
			}
			if val == "" {
				return fmt.Errorf("value is empty — pass --value or pipe on stdin")
			}
			mh, err := resolveMagicHome()
			if err != nil {
				return err
			}
			dir := core.NewPaths(mh).Secrets
			if err := secrets.SetProviderSecret(dir, keyEnv, val); err != nil {
				return err
			}
			fmt.Printf("set %s in %s\n", keyEnv, secrets.ProvidersPath(dir))
			return nil
		},
	}
	c.Flags().StringVar(&valueFlag, "value", "", "secret value (prefer over argv; else read stdin)")
	return c
}

func cmdSecretsList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List provider secret key names (no values)",
		Long:  `List key_env names in local plaintext secrets/providers.json. Values are never printed. No passphrase.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			mh, err := resolveMagicHome()
			if err != nil {
				return err
			}
			dir := core.NewPaths(mh).Secrets
			names, err := secrets.ListProviderSecretNames(dir)
			if err != nil {
				return err
			}
			if len(names) == 0 {
				fmt.Println("(no provider secrets — magic secrets set <key_env> --value …)")
				return nil
			}
			for _, n := range names {
				fmt.Println(n)
			}
			return nil
		},
	}
}
