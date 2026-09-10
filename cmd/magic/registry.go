package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"magic/internal/catalog"
	"magic/internal/core"
	"magic/internal/modelfile"
	"magic/internal/modelsdev"
	"magic/internal/openrouter"
)

func cmdRegistry() *cobra.Command {
	c := &cobra.Command{
		Use:   "registry",
		Short: "Model registry (local cache under $MAGIC_HOME/registry)",
	}
	c.AddCommand(cmdRegistrySync())
	c.AddCommand(cmdRegistrySearch())
	c.AddCommand(cmdRegistryShow())
	return c
}

func cmdRegistrySync() *cobra.Command {
	var apiURL string
	var source string
	c := &cobra.Command{
		Use:   "sync",
		Short: "Fetch model catalog source(s) and normalize into models.json",
		Long: `Fetch upstream model metadata into $MAGIC_HOME/registry/, then write
normalized Magic catalog models.json (Pi-shaped meta + thinkingLevelMap).

Sources:
  openrouter  (default)  slim OpenRouter /api/v1/models → openrouter-models.json
                         (also re-normalize from existing models.dev.json if present)
  models.dev             raw https://models.dev/api.json → models.dev.json + normalize
  all                    both raw fetches + normalize (models.dev primary; OR enrich)

See docs/model-registry-from-pi.md.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mh, err := resolveMagicHome()
			if err != nil {
				return err
			}
			src := strings.ToLower(strings.TrimSpace(source))
			if src == "" {
				src = "openrouter"
			}
			switch src {
			case "openrouter":
				if err := syncOpenRouter(mh, apiURL); err != nil {
					return err
				}
				// Optionally normalize from existing models.dev dump.
				return normalizeIfPossible(mh, true)
			case "models.dev", "modelsdev", "models-dev":
				if err := syncModelsDev(mh, apiURL); err != nil {
					return err
				}
				return normalizeRegistry(mh, false)
			case "all":
				if err := syncModelsDev(mh, ""); err != nil {
					return err
				}
				if err := syncOpenRouter(mh, apiURL); err != nil {
					return err
				}
				return normalizeRegistry(mh, true)
			default:
				return fmt.Errorf("unknown --source %q (want openrouter|models.dev|all)", source)
			}
		},
	}
	c.Flags().StringVar(&apiURL, "url", "", "override fetch URL for the selected source (tests / mirrors)")
	c.Flags().StringVar(&source, "source", "openrouter", "openrouter | models.dev | all")
	return c
}

func syncOpenRouter(mh, apiURL string) error {
	cache, err := openrouter.Sync(openrouter.SyncOptions{
		MagicHome: mh,
		APIURL:    apiURL,
	})
	if err != nil {
		return err
	}
	fmt.Printf("synced %d models → %s\n", len(cache.Models), openrouter.CachePath(mh))
	return nil
}

func syncModelsDev(mh, apiURL string) error {
	res, err := modelsdev.Sync(modelsdev.SyncOptions{
		MagicHome: mh,
		APIURL:    apiURL,
	})
	if err != nil {
		return err
	}
	fmt.Printf("synced models.dev (%d bytes) → %s\n", res.Bytes, res.Path)
	return nil
}

// normalizeRegistry reads models.dev.json (+ optional openrouter cache) and writes models.json.
func normalizeRegistry(mh string, enrichOpenRouter bool) error {
	path := modelsdev.CachePath(mh)
	rawCat, err := modelsdev.LoadCatalog(path)
	if err != nil {
		return fmt.Errorf("normalize: read %s: %w", path, err)
	}
	opts := modelfile.NormalizeOptions{
		ModelsDev: rawCat,
		SyncedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	if enrichOpenRouter {
		if or, err := openrouter.LoadCache(mh); err == nil {
			opts.OpenRouter = or
		}
	}
	cat, err := modelfile.Normalize(opts)
	if err != nil {
		return err
	}
	if err := modelfile.WriteCatalog(mh, cat); err != nil {
		return err
	}
	fmt.Printf("normalized %d models → %s\n", len(cat.Models), modelfile.CachePath(mh))
	return nil
}

// normalizeIfPossible re-normalizes when models.dev.json exists; no-op otherwise.
func normalizeIfPossible(mh string, enrichOpenRouter bool) error {
	if _, err := os.Stat(modelsdev.CachePath(mh)); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return normalizeRegistry(mh, enrichOpenRouter)
}

func cmdRegistrySearch() *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Substring search over normalized models.json (fallback: OpenRouter cache)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mh, err := resolveMagicHome()
			if err != nil {
				return err
			}
			if cat, err := modelfile.LoadCatalog(mh); err == nil {
				hits := modelfile.Search(cat, args[0])
				fmt.Print(modelfile.FormatSearchList(hits))
				return nil
			}
			cache, err := openrouter.LoadCache(mh)
			if err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("no registry cache — run: magic registry sync --source models.dev (or all)")
				}
				return err
			}
			hits := openrouter.Search(cache, args[0])
			fmt.Print(openrouter.FormatSearchList(hits))
			return nil
		},
	}
}

func cmdRegistryShow() *cobra.Command {
	return &cobra.Command{
		Use:   "show <model-id>",
		Short: "Print metadata for one model (normalized models.json preferred)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mh, err := resolveMagicHome()
			if err != nil {
				return err
			}
			if cat, err := modelfile.LoadCatalog(mh); err == nil {
				m, ok := modelfile.FindFuzzy(cat, args[0], "")
				if !ok {
					return fmt.Errorf("model %q not in models.json — try: magic registry sync --source all", args[0])
				}
				fmt.Print(modelfile.FormatModel(m))
				return nil
			}
			cache, err := openrouter.LoadCache(mh)
			if err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("no registry cache — run: magic registry sync --source models.dev (or all)")
				}
				return err
			}
			m, ok := openrouter.Find(cache, args[0])
			if !ok {
				return fmt.Errorf("model %q not in cache", args[0])
			}
			fmt.Print(openrouter.FormatModel(m))
			return nil
		},
	}
}

func cmdCatalogPick() *cobra.Command {
	return &cobra.Command{
		Use:   "pick <provider> <model-id>",
		Short: "Append model-id to provider pick in providers.yaml (optional models_meta from registry)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			provider, modelID := args[0], args[1]
			mh, err := resolveMagicHome()
			if err != nil {
				return err
			}
			path := core.NewPaths(mh).ProvidersYAML
			f, err := catalog.Load(path)
			if err != nil {
				return err
			}
			changed, err := f.AddPick(provider, modelID)
			if err != nil {
				return err
			}
			metaSet := false
			if cat, err := modelfile.LoadCatalog(mh); err == nil {
				if m, ok := modelfile.FindFuzzy(cat, modelID, provider); ok {
					f.SetModelMeta(modelID, catalog.ModelMeta{
						Name:          m.Name,
						ContextLength: m.ContextWindow,
						Source:        m.Source,
					})
					metaSet = true
				} else if m, ok := modelfile.FindFuzzy(cat, modelID, ""); ok {
					f.SetModelMeta(modelID, catalog.ModelMeta{
						Name:          m.Name,
						ContextLength: m.ContextWindow,
						Source:        m.Source,
					})
					metaSet = true
				}
			}
			if !metaSet {
				if cache, err := openrouter.LoadCache(mh); err == nil {
					if m, ok := openrouter.Find(cache, modelID); ok {
						f.SetModelMeta(modelID, catalog.ModelMeta{
							Name:          m.Name,
							ContextLength: m.ContextLength,
							Source:        "openrouter",
						})
					}
				}
			}
			if err := catalog.Save(path, f); err != nil {
				return err
			}
			if changed {
				fmt.Printf("added %s → providers.%s.pick\n", modelID, provider)
			} else {
				fmt.Printf("already in providers.%s.pick: %s\n", provider, modelID)
			}
			return nil
		},
	}
}
