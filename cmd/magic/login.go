package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"magic/internal/catalog"
	"magic/internal/login"
	"magic/internal/provider/builtin"
	"magic/internal/secrets"
)

func cmdLogin() *cobra.Command {
	var apiKey, pick string
	var yes, noLive bool
	c := &cobra.Command{
		Use:   "login [provider]",
		Short: "交互添加 provider：拉 /models、勾选、写 providers.yaml + 密钥",
		Long: `Pick a provider, enter its API key, fetch the live model list (GET /models),
select models, then write providers.yaml (pick) and the local plaintext
secret. Provider skeletons come from the builtin template.

  magic login deepseek                 # interactive TUI
  magic login                          # choose provider from a list
  magic login deepseek --api-key sk-…  # key from flag (still interactive pick)
  magic login deepseek --api-key sk-… --pick a,b --yes   # fully scripted

Fallbacks when GET /models is unavailable (always reported):
  1. the builtin template's default pick
  2. registry/models.json (run: magic registry sync --source models.dev)
  3. manual model-id entry

Existing providers are kept: user endpoints / key_env / models win; pick is
replaced with the selection.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mh, err := resolveMagicHome()
			if err != nil {
				return err
			}
			interactive := stdinIsTTY()

			name := ""
			if len(args) == 1 {
				name = strings.TrimSpace(args[0])
			} else {
				if !interactive {
					return fmt.Errorf("provider name required in non-interactive mode (see magic provider add --list)")
				}
				names, err := builtin.Names()
				if err != nil {
					return err
				}
				name, err = login.ChooseProvider("Choose a provider to log in", names)
				if err != nil {
					if login.IsAborted(err) {
						return nil
					}
					return err
				}
			}

			src, ok, err := login.LookupProvider(mh, name)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("unknown provider %q — see magic provider add --list", name)
			}

			key := strings.TrimSpace(apiKey)
			if key == "" {
				if v, ok, _ := secrets.GetProviderSecret(filepath.Join(mh, "secrets"), src.KeyEnv); ok {
					key = v
					fmt.Printf("using stored key for %s\n", src.KeyEnv)
				}
			}
			if key == "" {
				if !interactive {
					return fmt.Errorf("no key: pass --api-key or run magic secrets set %s", src.KeyEnv)
				}
				key, err = login.PromptKey(fmt.Sprintf("API key for %s", name), src.KeyEnv)
				if err != nil {
					if login.IsAborted(err) {
						return nil
					}
					return err
				}
			}

			plan, err := login.Resolve(login.Options{
				MagicHome: mh,
				Provider:  name,
				APIKey:    key,
				Pick:      parseIDList(pick),
				NoLive:    noLive,
			})
			if err != nil {
				return err
			}
			for _, n := range plan.Notes {
				fmt.Println("note:", n)
			}

			selected := plan.Selected
			if len(selected) == 0 {
				switch {
				case len(plan.Models) == 0 || plan.Source == "manual":
					if !interactive {
						return fmt.Errorf("no model list available — pass --pick a,b,c")
					}
					selected, err = login.PromptIDs(fmt.Sprintf("Model ids for %s (comma separated)", name))
				default:
					if !interactive {
						return fmt.Errorf("model selection needs a TTY — pass --pick a,b,c")
					}
					preselect := login.DefaultSelection(plan.Models, src.Pick)
					selected, err = login.SelectModels(fmt.Sprintf("Models for %s", name), plan.Source, plan.Models, preselect)
				}
				if err != nil {
					if login.IsAborted(err) {
						return nil
					}
					return err
				}
			}
			if len(selected) == 0 {
				return fmt.Errorf("no models selected")
			}

			summary := fmt.Sprintf("provider : %s\nkey_env  : %s\nendpoints: %s\nmodels   : %s",
				name, plan.KeyEnv, endpointsSummary(plan.Endpoints), strings.Join(selected, ", "))
			if !yes {
				if !interactive {
					return fmt.Errorf("confirmation needs a TTY — pass --yes")
				}
				ok, err := login.Confirm(summary)
				if err != nil {
					if login.IsAborted(err) {
						return nil
					}
					return err
				}
				if !ok {
					fmt.Println("cancelled")
					return nil
				}
			}

			path, err := login.Apply(mh, plan, selected)
			if err != nil {
				return err
			}
			fmt.Printf("wrote %s\n", path)
			fmt.Printf("stored %s in secrets/providers.json\n", plan.KeyEnv)
			return nil
		},
	}
	c.Flags().StringVar(&apiKey, "api-key", "", "API key (non-interactive)")
	c.Flags().StringVar(&pick, "pick", "", "comma-separated model ids (non-interactive selection)")
	c.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt")
	c.Flags().BoolVar(&noLive, "no-live", false, "skip GET /models; use builtin/registry fallback")
	return c
}

func stdinIsTTY() bool {
	fd := os.Stdin.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

func parseIDList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\n' || r == '\t'
	})
	var out []string
	seen := map[string]bool{}
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" || seen[f] {
			continue
		}
		seen[f] = true
		out = append(out, f)
	}
	return out
}

func endpointsSummary(eps []catalog.Endpoint) string {
	if len(eps) == 0 {
		return "(none)"
	}
	var parts []string
	for _, ep := range eps {
		parts = append(parts, fmt.Sprintf("%s(%s)", ep.Protocol, ep.BaseURL))
	}
	return strings.Join(parts, ", ")
}
