package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

// Status lists object count/prefix under vault/.
func Status(ctx context.Context, cfg Config) (string, error) {
	c := client(cfg)
	out, err := c.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(cfg.Bucket),
		Prefix: aws.String("vault/"),
	})
	if err != nil {
		return "", err
	}
	n := 0
	if out.KeyCount != nil {
		n = int(*out.KeyCount)
	}
	return fmt.Sprintf("bucket=%s prefix=vault/ objects≈%d", cfg.Bucket, n), nil
}

// Push uploads magic home files under vault/ prefix.
func Push(ctx context.Context, cfg Config, magicHome string) error {
	c := client(cfg)
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
		key := "vault/" + filepath.ToSlash(rel)
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = c.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(cfg.Bucket),
			Key:    aws.String(key),
			Body:   f,
		})
		return err
	})
}

// Pull downloads vault/ prefix into magic home.
func Pull(ctx context.Context, cfg Config, magicHome string) error {
	c := client(cfg)
	out, err := c.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(cfg.Bucket),
		Prefix: aws.String("vault/"),
	})
	if err != nil {
		return err
	}
	for _, obj := range out.Contents {
		if obj.Key == nil {
			continue
		}
		key := *obj.Key
		rel := strings.TrimPrefix(key, "vault/")
		if rel == "" || strings.HasSuffix(key, "/") {
			continue
		}
		dst := filepath.Join(magicHome, filepath.FromSlash(rel))
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
		f, err := os.Create(dst)
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
