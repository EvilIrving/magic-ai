package sync

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"magic/internal/secrets"
)

// Config from environment for Cloudflare R2 (S3-compatible).
type Config struct {
	Bucket    string
	AccessKey string
	SecretKey string
	Endpoint  string
	Region    string
}

// FromEnv loads R2 settings. Missing required fields → ok=false with message.
func FromEnv() (cfg Config, ok bool, msg string) {
	cfg.Bucket = os.Getenv("MAGIC_R2_BUCKET")
	cfg.AccessKey = firstEnv("MAGIC_R2_ACCESS_KEY_ID", "AWS_ACCESS_KEY_ID")
	cfg.SecretKey = firstEnv("MAGIC_R2_SECRET_ACCESS_KEY", "AWS_SECRET_ACCESS_KEY")
	cfg.Endpoint = os.Getenv("MAGIC_R2_ENDPOINT")
	cfg.Region = os.Getenv("MAGIC_R2_REGION")
	if cfg.Region == "" {
		cfg.Region = "auto"
	}
	if cfg.Bucket == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		msg = `R2 sync is not configured.

Set these environment variables:
  MAGIC_R2_BUCKET              — bucket name
  MAGIC_R2_ACCESS_KEY_ID       — access key (or AWS_ACCESS_KEY_ID)
  MAGIC_R2_SECRET_ACCESS_KEY   — secret key (or AWS_SECRET_ACCESS_KEY)
  MAGIC_R2_ENDPOINT            — optional, e.g. https://<accountid>.r2.cloudflarestorage.com
  MAGIC_R2_REGION              — optional, default "auto"

Then: magic sync status | push | pull`
		return cfg, false, msg
	}
	return cfg, true, ""
}

func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

func client(cfg Config) *s3.Client {
	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, _ ...any) (aws.Endpoint, error) {
		if cfg.Endpoint != "" {
			return aws.Endpoint{URL: cfg.Endpoint, HostnameImmutable: true}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})
	awsCfg := aws.Config{
		Region:                      cfg.Region,
		Credentials:                 credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		EndpointResolverWithOptions: resolver,
	}
	return s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})
}

func listVaultKeys(ctx context.Context, c *s3.Client, bucket string) ([]string, error) {
	var keys []string
	var token *string
	for {
		out, err := c.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(bucket),
			Prefix:            aws.String("vault/"),
			ContinuationToken: token,
		})
		if err != nil {
			return nil, err
		}
		for _, obj := range out.Contents {
			if obj.Key != nil && *obj.Key != "" {
				keys = append(keys, *obj.Key)
			}
		}
		if out.IsTruncated == nil || !*out.IsTruncated || out.NextContinuationToken == nil {
			break
		}
		token = out.NextContinuationToken
	}
	return keys, nil
}

// vaultDest maps object key vault/<rel> → path under magicHome. Rejects escapes.
func vaultDest(magicHome, key string) (string, bool) {
	if !strings.HasPrefix(key, "vault/") {
		return "", false
	}
	rel := strings.TrimPrefix(key, "vault/")
	if rel == "" || strings.HasSuffix(key, "/") {
		return "", false
	}
	home := filepath.Clean(magicHome)
	dst := filepath.Clean(filepath.Join(home, filepath.FromSlash(rel)))
	sep := string(os.PathSeparator)
	if dst != home && !strings.HasPrefix(dst, home+sep) {
		return "", false
	}
	return dst, true
}

// Status lists object count/prefix under vault/.
func Status(ctx context.Context, cfg Config) (string, error) {
	c := client(cfg)
	keys, err := listVaultKeys(ctx, c, cfg.Bucket)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("bucket=%s prefix=vault/ objects≈%d", cfg.Bucket, len(keys)), nil
}

// isSecretBearing reports whether a magic-home relative path holds credentials.
// Such files are encrypted on push and uploaded as `<rel>.enc`; pull decrypts
// them back to plaintext. `.enc` never exists locally (decisions.md D2).
func isSecretBearing(rel string) bool {
	rel = filepath.ToSlash(rel)
	if !strings.HasPrefix(rel, "secrets/") {
		return false
	}
	return strings.HasSuffix(rel, ".json")
}

// Push uploads magic home files under vault/ prefix.
// A file is encrypted when it lives under secrets/ or when its content contains
// a known provider secret value (docs/secrets-and-sync.md); encrypted objects are
// uploaded as `<rel>.enc`. Everything else is uploaded as plaintext.
func Push(ctx context.Context, cfg Config, magicHome, passphrase string) error {
	c := client(cfg)
	secretValues := loadSecretValues(magicHome)
	return filepath.Walk(magicHome, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == "backups" && filepath.Dir(path) == magicHome {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(magicHome, path)
		if err != nil {
			return err
		}
		relSlash := filepath.ToSlash(rel)
		key, pathEncrypt, skip := pushPlan(relSlash)
		if skip {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if pathEncrypt || containsAnySecret(data, secretValues) {
			if passphrase == "" {
				return fmt.Errorf("%s present but MAGIC_PASSPHRASE unset — export MAGIC_PASSPHRASE to encrypt on push", relSlash)
			}
			enc, err := secrets.Encrypt(data, passphrase)
			if err != nil {
				return err
			}
			_, err = c.PutObject(ctx, &s3.PutObjectInput{
				Bucket: aws.String(cfg.Bucket),
				Key:    aws.String("vault/" + relSlash + ".enc"),
				Body:   bytes.NewReader(enc),
			})
			return err
		}
		_, err = c.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(cfg.Bucket),
			Key:    aws.String(key),
			Body:   bytes.NewReader(data),
		})
		return err
	})
}

// loadSecretValues returns the local provider secret values used for content-based
// encryption detection. Missing store or short values are ignored.
func loadSecretValues(magicHome string) []string {
	m, err := secrets.LoadProviderSecrets(filepath.Join(magicHome, "secrets"))
	if err != nil {
		return nil
	}
	vals := make([]string, 0, len(m))
	for _, v := range m {
		if len(strings.TrimSpace(v)) >= 6 {
			vals = append(vals, v)
		}
	}
	return vals
}

// containsAnySecret reports whether data embeds any known secret value.
func containsAnySecret(data []byte, values []string) bool {
	for _, v := range values {
		if v != "" && bytes.Contains(data, []byte(v)) {
			return true
		}
	}
	return false
}

// pushPlan maps a magic-home relative path to its cloud object plan:
// skip local `.enc` state, encrypt secret-bearing files to `<rel>.enc`, upload
// everything else as plaintext.
func pushPlan(rel string) (key string, encrypt, skip bool) {
	if strings.HasSuffix(rel, ".enc") {
		return "", false, true
	}
	if isSecretBearing(rel) {
		return "vault/" + rel + ".enc", true, false
	}
	return "vault/" + rel, false, false
}

// Pull downloads vault/ prefix into magic home.
// `<rel>.enc` objects are decrypted to local plaintext (no `.enc` on disk);
// all other objects are written as-is.
func Pull(ctx context.Context, cfg Config, magicHome, passphrase string) error {
	c := client(cfg)
	keys, err := listVaultKeys(ctx, c, cfg.Bucket)
	if err != nil {
		return err
	}
	for _, key := range keys {
		if strings.HasSuffix(key, ".enc") {
			if passphrase == "" {
				return fmt.Errorf("%s present but MAGIC_PASSPHRASE unset — export MAGIC_PASSPHRASE to decrypt on pull", key)
			}
			dst, ok := vaultDest(magicHome, strings.TrimSuffix(key, ".enc"))
			if !ok {
				continue
			}
			blob, err := getObjectBytes(ctx, c, cfg.Bucket, key)
			if err != nil {
				return err
			}
			pt, err := secrets.Decrypt(blob, passphrase)
			if err != nil {
				return fmt.Errorf("decrypt %s: %w", key, err)
			}
			if err := writeLocalFile(dst, pt); err != nil {
				return err
			}
			continue
		}
		dst, ok := vaultDest(magicHome, key)
		if !ok {
			continue
		}
		res, err := c.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(cfg.Bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			res.Body.Close()
			return err
		}
		f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			res.Body.Close()
			return err
		}
		_, copyErr := f.ReadFrom(res.Body)
		res.Body.Close()
		closeErr := f.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func getObjectBytes(ctx context.Context, c *s3.Client, bucket, key string) ([]byte, error) {
	res, err := c.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return io.ReadAll(res.Body)
}

func writeLocalFile(dst string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp := dst + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}
