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
		Short: "Manage encrypted provider API keys (secrets/providers.json.enc)",
		Long: `Provider API keys for magic copy live in encrypted
$MAGIC_HOME/secrets/providers.json.enc, indexed by key_env from providers.yaml.

Requires MAGIC_PASSPHRASE for set/list (decrypt/encrypt). Adapter reinject
files (secrets/<adapterID>.json.enc) are separate and unchanged.`,
	}
	c.AddCommand(cmdSecretsSet())
	c.AddCommand(cmdSecretsList())
	return c
}

func cmdSecretsSet() *cobra.Command {
	var valueFlag string
	c := &cobra.Command{
		Use:   "set <key_env>",
		Short: "Set a provider API key in secrets/providers.json.enc",
		Long: `Merge <key_env> → value into encrypted secrets/providers.json.enc.

Prefer --value (or stdin) so shells do not store the secret in history:

  magic secrets set DEEPSEEK_API_KEY --value 'sk-…'
  echo 'sk-…' | magic secrets set DEEPSEEK_API_KEY
  magic secrets set DEEPSEEK_API_KEY   # reads value from stdin

Requires MAGIC_PASSPHRASE.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			keyEnv := strings.TrimSpace(args[0])
			if keyEnv == "" {
				return fmt.Errorf("key_env must be non-empty")
			}
			pass := os.Getenv("MAGIC_PASSPHRASE")
			if pass == "" {
				return fmt.Errorf("MAGIC_PASSPHRASE required to set provider secrets")
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
			if err := secrets.SetProviderSecret(dir, pass, keyEnv, val); err != nil {
				return err
			}
			fmt.Printf("set %s in %s/%s\n", keyEnv, dir, secrets.ProvidersFileName)
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
		Long:  `List key_env names stored in secrets/providers.json.enc. Values are never printed. Requires MAGIC_PASSPHRASE.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pass := os.Getenv("MAGIC_PASSPHRASE")
			mh, err := resolveMagicHome()
			if err != nil {
				return err
			}
			dir := core.NewPaths(mh).Secrets
			// If file missing, Load returns empty; passphrase may be empty.
			// If file exists, passphrase required (LoadProviderSecrets errors).
			names, err := secrets.ListProviderSecretNames(dir, pass)
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
