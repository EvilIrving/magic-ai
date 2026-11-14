package core

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"magic/internal/backup"
	"magic/internal/writefile"
)

// Migration is one magic.yaml field `magic migrate` changes. Field is the YAML
// key, From and To the rendered values; the terminal prints one line per entry
// and --dry-run prints the same list without touching the file.
type Migration struct {
	Field string `json:"field"`
	From  string `json:"from"`
	To    string `json:"to"`
}

// MigrateOptions controls MigrateLibrary.
type MigrateOptions struct {
	MagicHome string
	// DryRun lists the migrations without backing up or writing magic.yaml.
	DryRun bool
}

// MigrateResult is what one migrate run found and did.
type MigrateResult struct {
	// SchemaVersion is the version magic.yaml had when migrate started.
	SchemaVersion int `json:"schema_version"`
	// TargetVersion is the version this magic writes (LibrarySchemaVersion).
	TargetVersion int         `json:"target_version"`
	Changes       []Migration `json:"changes,omitempty"`
	// BackupDir is the set magic.yaml was copied into before the write.
	BackupDir string `json:"backup_dir,omitempty"`
	// AlreadyCurrent: the library was at TargetVersion; no backup, no write.
	AlreadyCurrent bool `json:"already_current"`
	DryRun         bool `json:"dry_run"`
}

// MigrateLibrary upgrades magic.yaml to LibrarySchemaVersion.
//
// The file is edited as a yaml.Node tree, not re-marshalled from MagicConfig: the
// user's other fields, their order and their comments stay exactly where they
// are, and only schema_version is added or changed. Before the write the current
// file is copied into backups/<stamp>/ through internal/backup, so the migration
// can be rolled back like any other library write.
//
// It is idempotent: a library already at LibrarySchemaVersion returns
// AlreadyCurrent without a backup or a write, and a library written by a newer
// magic is refused instead of downgraded.
func MigrateLibrary(opts MigrateOptions) (MigrateResult, error) {
	path := NewPaths(opts.MagicHome).MagicYAML
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return MigrateResult{}, fmt.Errorf("magic.yaml 不存在: %s（运行 magic init 创建配置库）", path)
		}
		return MigrateResult{}, err
	}
	doc, root, err := parseMagicYAMLDocument(raw)
	if err != nil {
		return MigrateResult{}, fmt.Errorf("magic.yaml 无法解析: %w", err)
	}
	from, err := magicSchemaVersion(root)
	if err != nil {
		return MigrateResult{}, fmt.Errorf("magic.yaml: %w", err)
	}
	res := MigrateResult{SchemaVersion: from, TargetVersion: LibrarySchemaVersion, DryRun: opts.DryRun}
	if res.SchemaVersion > LibrarySchemaVersion {
		return res, fmt.Errorf("库结构版本 %d 比 magic 支持的 %d 新，拒绝写入（请升级 magic）", res.SchemaVersion, LibrarySchemaVersion)
	}
	if res.SchemaVersion == LibrarySchemaVersion {
		res.AlreadyCurrent = true
		return res, nil
	}

	setMagicSchemaVersion(root, LibrarySchemaVersion)
	out, err := encodeMagicYAMLDocument(doc)
	if err != nil {
		return res, err
	}
	res.Changes = append(res.Changes, Migration{
		Field: "schema_version",
		From:  strconv.Itoa(from),
		To:    strconv.Itoa(LibrarySchemaVersion),
	})
	if opts.DryRun {
		return res, nil
	}

	dir, err := NewBackupDir(opts.MagicHome)
	if err != nil {
		return res, err
	}
	res.BackupDir = dir
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(path); statErr == nil {
		mode = info.Mode().Perm()
	}
	if err := backup.Record(dir, backup.Entry{
		Adapter: "library",
		Rel:     "magic.yaml",
		Target:  path,
		Backup:  "magic.yaml",
		Action:  backup.ActionUpdate,
		Mode:    backup.ModeString(mode),
		After:   backup.FingerprintOf(out),
	}); err != nil {
		return res, err
	}
	if err := writefile.Atomic(path, out, mode); err != nil {
		return res, err
	}
	return res, nil
}

// parseMagicYAMLDocument parses magic.yaml into a document whose root is a
// mapping. An empty file is the legacy library without any field: migrate gives
// it a mapping root so schema_version can be added.
func parseMagicYAMLDocument(raw []byte) (*yaml.Node, *yaml.Node, error) {
	if strings.TrimSpace(string(raw)) == "" {
		root := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		return &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{root}}, root, nil
	}
	doc := &yaml.Node{}
	if err := yaml.Unmarshal(raw, doc); err != nil {
		return nil, nil, err
	}
	root := magicYAMLRoot(doc)
	if root == nil || root.Kind != yaml.MappingNode {
		return nil, nil, fmt.Errorf("根节点不是 mapping，无法确定字段顺序")
	}
	return doc, root, nil
}

func magicYAMLRoot(doc *yaml.Node) *yaml.Node {
	if doc == nil {
		return nil
	}
	if doc.Kind == yaml.DocumentNode {
		if len(doc.Content) == 0 {
			return nil
		}
		return doc.Content[0]
	}
	return doc
}

// magicSchemaVersion reads schema_version from the mapping. A missing key is 0:
// the library predates the field.
func magicSchemaVersion(root *yaml.Node) (int, error) {
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value != "schema_version" {
			continue
		}
		value := root.Content[i+1]
		if value.Kind != yaml.ScalarNode {
			return 0, fmt.Errorf("schema_version 不是标量")
		}
		n, err := strconv.Atoi(strings.TrimSpace(value.Value))
		if err != nil {
			return 0, fmt.Errorf("schema_version %q 不是整数", value.Value)
		}
		return n, nil
	}
	return 0, nil
}

// setMagicSchemaVersion writes schema_version, keeping the user's field order:
// an existing key keeps its position and comments, a new one goes right after
// `version` (where magic init writes it) or first when there is no version key.
func setMagicSchemaVersion(root *yaml.Node, version int) {
	value := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.Itoa(version)}
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value != "schema_version" {
			continue
		}
		old := root.Content[i+1]
		value.HeadComment, value.LineComment, value.FootComment = old.HeadComment, old.LineComment, old.FootComment
		root.Content[i+1] = value
		return
	}
	key := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "schema_version"}
	at := 0
	if len(root.Content) >= 2 && root.Content[0].Value == "version" {
		at = 2
	} else if len(root.Content) >= 1 && root.Content[0].HeadComment != "" {
		// The leading comment is the file's header, not the first key's: keep it
		// on top when schema_version becomes the first key.
		key.HeadComment, root.Content[0].HeadComment = root.Content[0].HeadComment, ""
	}
	content := make([]*yaml.Node, 0, len(root.Content)+2)
	content = append(content, root.Content[:at]...)
	content = append(content, key, value)
	content = append(content, root.Content[at:]...)
	root.Content = content
}

// encodeMagicYAMLDocument renders the document with magic.yaml's two-space
// indent. yaml.v3 keeps every node's comments and order.
func encodeMagicYAMLDocument(doc *yaml.Node) ([]byte, error) {
	var b strings.Builder
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return []byte(b.String()), nil
}
