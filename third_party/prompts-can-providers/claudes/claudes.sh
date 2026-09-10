#!/usr/bin/env python3
r"""Self-contained Claude Code provider switcher.

This file is the provider catalog, command-line tool, installer, and user guide.
It does not need a repository-side models.json.

Supported Claude Code routes:
  - Anthropic official login
  - DeepSeek, Kimi, and MiniMax official APIs
  - api.shu.cool OpenAI, DeepSeek, and Grok models

OpenAI and Grok subscription logins are managed by gpts and groks. A product
subscription is not an API credential that Claude Code can reuse directly.

Quick start:
  ./claudes.sh list
  ./claudes.sh env deepseek
  ./claudes.sh use deepseek
  ./claudes.sh status

Install:
  ./claudes.sh install
  source ~/.zshrc
  claudes help

Windows:
  powershell -ExecutionPolicy Bypass -File .\claudes.ps1 install

Legacy migration:
  claudes migrate

The migration command imports non-empty keys from the old
~/.claude/models.json into shell environment entries. The legacy file is never
deleted automatically.
"""

from __future__ import annotations

import argparse
import copy
import getpass
import json
import os
import re
import shlex
import shutil
import stat
import sys
import tempfile
import uuid
from contextlib import contextmanager
from dataclasses import dataclass, field
from datetime import datetime
from pathlib import Path
from typing import Any, Iterator

TOOL_VERSION = "2.0.0"
DEFAULT_BACKUP_LIMIT = 3
MODEL_ENV_KEYS = {
    "haiku": "ANTHROPIC_DEFAULT_HAIKU_MODEL",
    "sonnet": "ANTHROPIC_DEFAULT_SONNET_MODEL",
    "opus": "ANTHROPIC_DEFAULT_OPUS_MODEL",
    "default": "ANTHROPIC_MODEL",
    "reasoning": "ANTHROPIC_REASONING_MODEL",
}
MANAGED_ENV_KEYS = {
    "ANTHROPIC_API_KEY",
    "ANTHROPIC_AUTH_TOKEN",
    "ANTHROPIC_BASE_URL",
    "ANTHROPIC_DEFAULT_FABLE_MODEL",
    *MODEL_ENV_KEYS.values(),
}


@dataclass(frozen=True)
class Provider:
    name: str
    base_url: str = ""
    key_env: str = ""
    models: dict[str, str] = field(default_factory=dict)


PROVIDERS = {
    "official": Provider(name="Anthropic Official"),
    "deepseek": Provider(
        name="DeepSeek Official API",
        base_url="https://api.deepseek.com/anthropic",
        key_env="DEEPSEEK_API_KEY",
        models={
            "haiku": "deepseek-v4-flash-vision-exp",
            "sonnet": "deepseek-v4-pro",
            "opus": "deepseek-v4-pro",
            "default": "deepseek-v4-pro",
        },
    ),
    "kimi": Provider(
        name="Kimi Official API",
        base_url="https://api.moonshot.cn/anthropic",
        key_env="KIMI_API_KEY",
        models={
            "haiku": "kimi-k2.5",
            "sonnet": "kimi-k2.5",
            "opus": "kimi-k2.5",
            "default": "kimi-k2.5",
            "reasoning": "kimi-k2.5",
        },
    ),
    "minimax": Provider(
        name="MiniMax Official API",
        base_url="https://api.minimaxi.com/anthropic",
        key_env="MINIMAX_API_KEY",
        models={
            "haiku": "MiniMax-M2.7",
            "sonnet": "MiniMax-M2.7",
            "opus": "MiniMax-M2.7",
            "default": "MiniMax-M2.7",
        },
    ),
    "s-openai": Provider(
        name="Shu OpenAI",
        base_url="https://api.shu.cool",
        key_env="SHU_API_KEY",
        models={
            "haiku": "gpt-5.6-luna",
            "sonnet": "gpt-5.6-terra",
            "opus": "gpt-5.6-sol",
            "default": "gpt-5.6-sol",
        },
    ),
    "s-deepseek": Provider(
        name="Shu DeepSeek",
        base_url="https://api.shu.cool",
        key_env="SHU_DEEPSEEK_API_KEY",
        models={
            "haiku": "deepseek-v4-flash-vision-exp",
            "sonnet": "deepseek-v4-pro",
            "opus": "deepseek-v4-pro",
            "default": "deepseek-v4-pro",
        },
    ),
    "s-grok": Provider(
        name="Shu Grok",
        base_url="https://api.shu.cool",
        key_env="SHU_GROK_API_KEY",
        models={
            "haiku": "grok-composer-2.5-fast",
            "sonnet": "grok-4.5",
            "opus": "grok-4.5",
            "default": "grok-4.5",
        },
    ),
}
PROVIDER_ALIASES = {
    "anthropic": "official",
    "shu-openai": "s-openai",
    "shu-deepseek": "s-deepseek",
    "shu-grok": "s-grok",
    "shu": "s-openai",
}
KEY_MARKERS = {
    "SHU_API_KEY": "setup-shu: SHU_API_KEY",
    "SHU_DEEPSEEK_API_KEY": "claudes: SHU_DEEPSEEK_API_KEY",
    "SHU_GROK_API_KEY": "groks: SHU_GROK_API_KEY",
    "DEEPSEEK_API_KEY": "claudes: DEEPSEEK_API_KEY",
    "KIMI_API_KEY": "claudes: KIMI_API_KEY",
    "MINIMAX_API_KEY": "claudes: MINIMAX_API_KEY",
}
LEGACY_KEY_PROVIDERS = {
    "DEEPSEEK_API_KEY": ("deepseek",),
    "KIMI_API_KEY": ("kimi",),
    "MINIMAX_API_KEY": ("minimax",),
    "SHU_API_KEY": ("s-openai", "shu-openai"),
    "SHU_DEEPSEEK_API_KEY": ("s-deepseek", "shu-deepseek"),
    "SHU_GROK_API_KEY": ("s-grok", "shu-grok"),
}


class ClaudesError(Exception):
    pass


def claude_home() -> Path:
    return Path(os.environ.get("CLAUDE_HOME", Path.home() / ".claude")).expanduser()


def settings_file() -> Path:
    return claude_home() / "settings.json"


def legacy_models_file() -> Path:
    return Path(os.environ.get("CLAUDE_MODELS_FILE", claude_home() / "models.json")).expanduser()


def backup_dir() -> Path:
    return claude_home() / "provider-switch-backups"


def lock_dir() -> Path:
    return claude_home() / ".claudes.lock"


def profile_home() -> Path:
    return Path(os.environ.get("HOME", Path.home())).expanduser()


def profile_paths() -> list[Path]:
    home = profile_home()
    shell = os.environ.get("SHELL", "")
    preferred = home / (".bash_profile" if shell.endswith("bash") else ".zshrc")
    result = [preferred, home / ".zshrc", home / ".bash_profile", home / ".bashrc"]
    return list(dict.fromkeys(result))


def preferred_profile() -> Path:
    return profile_paths()[0]


def load_json(path: Path, *, missing_ok: bool = False) -> dict[str, Any]:
    if not path.is_file():
        if missing_ok:
            return {}
        raise ClaudesError(f"Missing file: {path}")
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        raise ClaudesError(f"Cannot read valid JSON from {path}: {exc}") from exc
    if not isinstance(value, dict):
        raise ClaudesError(f"Expected a JSON object in {path}")
    return value


def atomic_write_json(path: Path, data: dict[str, Any], *, mode: int = 0o600) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    descriptor, temp_name = tempfile.mkstemp(prefix=f".{path.name}.", suffix=".tmp", dir=path.parent)
    temp_path = Path(temp_name)
    try:
        with os.fdopen(descriptor, "w", encoding="utf-8") as handle:
            json.dump(data, handle, ensure_ascii=False, indent=2)
            handle.write("\n")
            handle.flush()
            os.fsync(handle.fileno())
        os.chmod(temp_path, mode)
        os.replace(temp_path, path)
    finally:
        temp_path.unlink(missing_ok=True)


def atomic_write_text(path: Path, text: str, *, mode: int | None = None) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    descriptor, temp_name = tempfile.mkstemp(prefix=f".{path.name}.", suffix=".tmp", dir=path.parent)
    temp_path = Path(temp_name)
    try:
        with os.fdopen(descriptor, "w", encoding="utf-8") as handle:
            handle.write(text)
            handle.flush()
            os.fsync(handle.fileno())
        if mode is None:
            mode = stat.S_IMODE(path.stat().st_mode) if path.exists() else 0o600
        os.chmod(temp_path, mode)
        os.replace(temp_path, path)
    finally:
        temp_path.unlink(missing_ok=True)


@contextmanager
def config_lock() -> Iterator[None]:
    claude_home().mkdir(parents=True, exist_ok=True)
    try:
        lock_dir().mkdir()
    except FileExistsError as exc:
        raise ClaudesError(f"Another claudes process is running: {lock_dir()}") from exc
    try:
        yield
    finally:
        try:
            lock_dir().rmdir()
        except FileNotFoundError:
            pass


def resolve_provider(value: str) -> str:
    provider_id = PROVIDER_ALIASES.get(value.lower(), value.lower())
    if provider_id not in PROVIDERS:
        available = ", ".join(PROVIDERS)
        raise ClaudesError(f"Unknown provider '{value}'. Available: {available}")
    return provider_id


def provider_for_key(value: str) -> Provider:
    normalized = value.lower()
    if normalized in {"shu", "company"}:
        return PROVIDERS["s-openai"]
    provider_id = resolve_provider(normalized)
    provider = PROVIDERS[provider_id]
    if not provider.key_env:
        raise ClaudesError("The official provider does not use a stored third-party API key")
    return provider


def parse_exported_key(line: str, key_env: str) -> str | None:
    try:
        tokens = shlex.split(line, comments=True, posix=True)
    except ValueError:
        return None
    if len(tokens) != 2 or tokens[0] != "export":
        return None
    prefix = f"{key_env}="
    if not tokens[1].startswith(prefix):
        return None
    value = tokens[1][len(prefix) :]
    return value or None


def persistent_key(key_env: str) -> str | None:
    value = os.environ.get(key_env, "").strip()
    if value:
        return value
    for path in profile_paths():
        if not path.is_file():
            continue
        for line in reversed(path.read_text(encoding="utf-8", errors="replace").splitlines()):
            value = parse_exported_key(line, key_env)
            if value:
                return value
    return None


def validate_key(value: str) -> str:
    key = value.strip()
    if not key:
        raise ClaudesError("API key cannot be empty; use 'clear' to remove it")
    if any(character in key for character in "\r\n\0") or any(ord(character) < 32 for character in key):
        raise ClaudesError("API key contains an invalid control character")
    return key


def is_managed_key_line(line: str, key_env: str) -> bool:
    marker = KEY_MARKERS[key_env]
    return bool(re.match(rf"^\s*export\s+{re.escape(key_env)}=", line)) and f"# {marker}" in line


def remove_managed_key_lines(path: Path, key_env: str) -> bool:
    if not path.is_file():
        return False
    original = path.read_text(encoding="utf-8", errors="replace")
    lines = original.splitlines(keepends=True)
    updated = "".join(line for line in lines if not is_managed_key_line(line, key_env))
    if updated == original:
        return False
    atomic_write_text(path, updated)
    return True


def write_profile_key(key_env: str, value: str, *, dry_run: bool = False) -> Path:
    target = preferred_profile()
    if dry_run:
        return target
    for path in profile_paths():
        remove_managed_key_lines(path, key_env)
    target.parent.mkdir(parents=True, exist_ok=True)
    if not target.exists():
        atomic_write_text(target, "", mode=0o600)
    existing = target.read_text(encoding="utf-8", errors="replace")
    separator = "" if not existing or existing.endswith("\n") else "\n"
    line = f"export {key_env}={shlex.quote(value)}  # {KEY_MARKERS[key_env]}\n"
    atomic_write_text(target, f"{existing}{separator}{line}")
    return target


def clear_profile_key(key_env: str, *, dry_run: bool = False) -> list[Path]:
    changed: list[Path] = []
    for path in profile_paths():
        if dry_run:
            if path.is_file() and any(
                is_managed_key_line(line, key_env)
                for line in path.read_text(encoding="utf-8", errors="replace").splitlines()
            ):
                changed.append(path)
        elif remove_managed_key_lines(path, key_env):
            changed.append(path)
    return changed


def create_backup(path: Path, backup_limit: int) -> Path | None:
    if not path.is_file():
        return None
    target_dir = backup_dir()
    target_dir.mkdir(parents=True, exist_ok=True)
    stamp = datetime.now().strftime("%Y%m%d-%H%M%S-%f")[:-3]
    target = target_dir / f"settings.{stamp}-{uuid.uuid4().hex[:8]}.json"
    shutil.copy2(path, target)
    os.chmod(target, 0o600)
    trim_backups(backup_limit)
    return target


def backup_files() -> list[Path]:
    if not backup_dir().is_dir():
        return []
    return sorted(
        backup_dir().glob("settings.*.json"),
        key=lambda item: item.stat().st_mtime_ns,
        reverse=True,
    )


def trim_backups(limit: int) -> None:
    for expired in backup_files()[limit:]:
        expired.unlink()


def build_settings(current: dict[str, Any], provider_id: str, api_key: str | None) -> dict[str, Any]:
    provider = PROVIDERS[provider_id]
    result = copy.deepcopy(current)
    env = result.get("env", {})
    if not isinstance(env, dict):
        raise ClaudesError("settings.json field 'env' must be an object")
    env = {str(key): value for key, value in env.items() if key not in MANAGED_ENV_KEYS}
    if provider_id != "official":
        if not api_key:
            raise ClaudesError(f"{provider.key_env} is not set. Run: claudes env {provider_id}")
        env["ANTHROPIC_BASE_URL"] = provider.base_url
        env["ANTHROPIC_AUTH_TOKEN"] = api_key
        for role, model in provider.models.items():
            if role in MODEL_ENV_KEYS:
                env[MODEL_ENV_KEYS[role]] = model
    result["env"] = env
    return result


def detect_provider(settings: dict[str, Any]) -> str:
    env = settings.get("env", {})
    if not isinstance(env, dict):
        return "unknown"
    base_url = str(env.get("ANTHROPIC_BASE_URL", "")).rstrip("/")
    if not base_url:
        return "official"
    matches = [
        provider_id
        for provider_id, provider in PROVIDERS.items()
        if provider.base_url.rstrip("/") == base_url
    ]
    if len(matches) == 1:
        return matches[0]
    model = str(env.get("ANTHROPIC_MODEL", "")).strip()
    model_matches = [
        provider_id for provider_id in matches if PROVIDERS[provider_id].models.get("default") == model
    ]
    return model_matches[0] if len(model_matches) == 1 else "unknown"


def use_provider(provider_value: str, *, backup_limit: int, dry_run: bool) -> None:
    provider_id = resolve_provider(provider_value)
    provider = PROVIDERS[provider_id]
    api_key = persistent_key(provider.key_env) if provider.key_env else None
    if dry_run:
        current = load_json(settings_file(), missing_ok=True)
        updated = build_settings(current, provider_id, api_key)
        if updated == current:
            if not settings_are_private(settings_file()):
                print(f"[DryRun] Would set permissions to 0600: {settings_file()}")
                return
            print(f"Already active: {provider_id}")
            return
        print(f"[DryRun] Would switch to: {provider_id}")
        print(f"[DryRun] Would update: {settings_file()}")
        return
    with config_lock():
        current = load_json(settings_file(), missing_ok=True)
        updated = build_settings(current, provider_id, api_key)
        if updated == current:
            if not settings_are_private(settings_file()):
                os.chmod(settings_file(), 0o600)
                print(f"Permissions fixed: {settings_file()} (0600)")
                return
            print(f"Already active: {provider_id}")
            return
        saved = create_backup(settings_file(), backup_limit)
        atomic_write_json(settings_file(), updated)
    if saved:
        print(f"Backup: {saved}")
    print(f"Active provider: {provider_id}")
    print("Restart Claude Code for the change to take effect.")


def clear_active_auth(key_env: str, *, backup_limit: int, dry_run: bool) -> Path | None:
    path = settings_file()
    if not path.is_file():
        return None
    current = load_json(path)
    active = detect_provider(current)
    provider = PROVIDERS.get(active)
    env = current.get("env", {})
    if not provider or provider.key_env != key_env or not isinstance(env, dict):
        return None
    if not env.get("ANTHROPIC_AUTH_TOKEN") and not env.get("ANTHROPIC_API_KEY"):
        return None
    if dry_run:
        print(f"[DryRun] Would remove the active {key_env} credential from {path}.")
        return None
    with config_lock():
        current = load_json(path)
        active = detect_provider(current)
        provider = PROVIDERS.get(active)
        env = current.get("env", {})
        if not provider or provider.key_env != key_env or not isinstance(env, dict):
            return None
        updated = copy.deepcopy(current)
        updated_env = updated["env"]
        updated_env.pop("ANTHROPIC_AUTH_TOKEN", None)
        updated_env.pop("ANTHROPIC_API_KEY", None)
        saved = create_backup(path, backup_limit)
        atomic_write_json(path, updated)
    print(f"Removed active credential from: {path}")
    return saved


def set_key(
    provider_value: str,
    value: str | None,
    *,
    backup_limit: int,
    dry_run: bool,
) -> None:
    provider = provider_for_key(provider_value)
    key_env = provider.key_env
    if value is not None and value.lower() == "clear":
        changed = clear_profile_key(key_env, dry_run=dry_run)
        prefix = "[DryRun] Would clear" if dry_run else "Cleared"
        print(f"{prefix}: {key_env}")
        if changed:
            print("Profiles: " + ", ".join(str(path) for path in changed))
        saved = clear_active_auth(key_env, backup_limit=backup_limit, dry_run=dry_run)
        if saved:
            print(f"Backup: {saved}")
        if os.environ.get(key_env):
            print(f"Open a new shell to discard the inherited {key_env} value.")
        return
    if value is None:
        value = getpass.getpass(f"{key_env} (input hidden): ")
    else:
        print("Warning: the API key may be present in shell history.", file=sys.stderr)
    key = validate_key(value)
    target = write_profile_key(key_env, key, dry_run=dry_run)
    prefix = "[DryRun] Would set" if dry_run else "Set"
    print(f"{prefix}: {key_env}")
    print(f"Profile: {target}")
    if not dry_run:
        print(f"Run: source {target}")


def show_list() -> None:
    settings = load_json(settings_file(), missing_ok=True)
    active = detect_provider(settings)
    for provider_id, provider in PROVIDERS.items():
        marker = "*" if provider_id == active else " "
        key_state = "not required" if not provider.key_env else ("set" if persistent_key(provider.key_env) else "missing")
        model = provider.models.get("default", "Claude default")
        print(f"{marker} {provider_id:<14} {model:<26} {provider.name} [{key_state}]")


def settings_permissions(path: Path) -> str:
    if not path.exists():
        return "missing"
    mode = stat.S_IMODE(path.stat().st_mode)
    return f"{mode:04o} ({'private' if mode & 0o077 == 0 else 'too open'})"


def settings_are_private(path: Path) -> bool:
    return path.is_file() and stat.S_IMODE(path.stat().st_mode) == 0o600


def show_status() -> None:
    path = settings_file()
    settings = load_json(path, missing_ok=True)
    active = detect_provider(settings) if path.is_file() else "missing settings"
    provider = PROVIDERS.get(active)
    env = settings.get("env", {}) if isinstance(settings.get("env", {}), dict) else {}
    key_state = "not required"
    if provider and provider.key_env:
        key_state = "set" if persistent_key(provider.key_env) else "missing"
    print("===== Claude Code provider status =====")
    print(f"Settings file : {path}")
    print(f"Permissions   : {settings_permissions(path)}")
    print(f"Provider      : {active}")
    print(f"Default model : {env.get('ANTHROPIC_MODEL', 'Claude default')}")
    print(f"Base URL      : {env.get('ANTHROPIC_BASE_URL', 'Anthropic official')}")
    print(f"Persistent key: {key_state}")
    print(f"Auth in config: {'yes' if env.get('ANTHROPIC_AUTH_TOKEN') else 'no'}")


def migrate_legacy_keys(*, dry_run: bool) -> None:
    path = legacy_models_file()
    data = load_json(path)
    providers = data.get("providers", {})
    if not isinstance(providers, dict):
        raise ClaudesError("Legacy models.json field 'providers' must be an object")
    migrated: list[str] = []
    skipped: list[str] = []
    pending: dict[str, str] = {}
    for key_env, provider_ids in LEGACY_KEY_PROVIDERS.items():
        if persistent_key(key_env):
            skipped.append(key_env)
            continue
        values = {
            str(providers[provider_id].get("apiKey", "")).strip()
            for provider_id in provider_ids
            if isinstance(providers.get(provider_id), dict)
            and str(providers[provider_id].get("apiKey", "")).strip()
        }
        if not values:
            continue
        if len(values) > 1:
            raise ClaudesError(f"Legacy providers contain conflicting values for {key_env}")
        pending[key_env] = validate_key(values.pop())
    for key_env, value in pending.items():
        write_profile_key(key_env, value, dry_run=dry_run)
        migrated.append(key_env)
    prefix = "[DryRun] Would migrate" if dry_run else "Migrated"
    print(f"{prefix}: {', '.join(migrated) if migrated else 'no keys'}")
    if skipped:
        print("Already configured: " + ", ".join(skipped))
    if not dry_run and migrated:
        print(f"Run: source {preferred_profile()}")
    print(f"Legacy file preserved: {path}")


def manage_backups(action: str, *, backup_limit: int, dry_run: bool) -> None:
    files = backup_files()
    if action == "list":
        if not files:
            print("No backups.")
        for path in files:
            print(path)
        return
    if action == "clear":
        if dry_run:
            print(f"[DryRun] Would remove {len(files)} backup(s).")
            return
        for path in files:
            path.unlink()
        if backup_dir().is_dir() and not any(backup_dir().iterdir()):
            backup_dir().rmdir()
        print(f"Removed {len(files)} backup(s).")
        return
    if not files:
        raise ClaudesError("No configuration backup is available to restore")
    source = files[0]
    restored = load_json(source)
    if dry_run:
        print(f"[DryRun] Would restore: {source}")
        return
    with config_lock():
        create_backup(settings_file(), backup_limit)
        atomic_write_json(settings_file(), restored)
    print(f"Restored: {source}")


def update_alias(profile: Path, target: Path) -> None:
    existing = profile.read_text(encoding="utf-8", errors="replace") if profile.is_file() else ""
    lines = [
        line
        for line in existing.splitlines()
        if not re.match(r"^\s*alias\s+(claudes|cc-provider)=", line)
    ]
    lines.append(f'alias claudes="{target}"')
    atomic_write_text(profile, "\n".join(lines).rstrip() + "\n")


def install() -> None:
    source = Path(__file__).resolve()
    target = claude_home() / "claudes"
    target.parent.mkdir(parents=True, exist_ok=True)
    if source != target.resolve():
        shutil.copy2(source, target)
    os.chmod(target, 0o755)
    profile = preferred_profile()
    if not profile.exists():
        atomic_write_text(profile, "", mode=0o600)
    update_alias(profile, target)
    print(f"Installed: {target}")
    print(f"Alias updated: {profile}")
    if legacy_models_file().is_file():
        print("Legacy keys found. Run after reloading the shell: claudes migrate")
    print(f"Run: source {profile}")


def show_help() -> None:
    print(
        f"""claudes v{TOOL_VERSION} - self-contained Claude Code provider switcher

Commands:
  claudes list
  claudes status
  claudes use <provider>
  claudes env <provider|shu> [clear]
  claudes migrate
  claudes backup list|restore|clear
  claudes install
  claudes help

Providers:
  official, deepseek, kimi, minimax, s-openai, s-deepseek, s-grok

API key environment variables:
  DEEPSEEK_API_KEY, KIMI_API_KEY, MINIMAX_API_KEY
  SHU_API_KEY, SHU_DEEPSEEK_API_KEY, SHU_GROK_API_KEY

Options:
  --claude-home <path>  --backup-limit <count>  --dry-run

Only provider-owned env fields in ~/.claude/settings.json are changed.
Existing settings are preserved and every real change creates a backup."""
    )


def parser() -> argparse.ArgumentParser:
    result = argparse.ArgumentParser(add_help=False)
    result.add_argument("--claude-home", "-ClaudeHome")
    result.add_argument("--backup-limit", "-BackupLimit", type=int, default=DEFAULT_BACKUP_LIMIT)
    result.add_argument("--dry-run", "-DryRun", action="store_true")
    subparsers = result.add_subparsers(dest="command")
    subparsers.add_parser("status", add_help=False)
    subparsers.add_parser("list", add_help=False)
    subparsers.add_parser("help", add_help=False)
    subparsers.add_parser("install", add_help=False)
    subparsers.add_parser("migrate", add_help=False)
    use = subparsers.add_parser("use", add_help=False)
    use.add_argument("provider")
    key = subparsers.add_parser("env", aliases=["key"], add_help=False)
    key.add_argument("provider")
    key.add_argument("value", nargs="?")
    backups = subparsers.add_parser("backup", add_help=False)
    backups.add_argument("action", choices=("list", "restore", "clear"))
    return result


def main() -> None:
    args = parser().parse_args()
    if args.claude_home:
        os.environ["CLAUDE_HOME"] = args.claude_home
    if args.backup_limit < 1:
        raise ClaudesError("--backup-limit must be at least 1")
    command = args.command or "status"
    if command == "status":
        show_status()
    elif command == "list":
        show_list()
    elif command == "use":
        use_provider(args.provider, backup_limit=args.backup_limit, dry_run=args.dry_run)
    elif command in {"env", "key"}:
        set_key(
            args.provider,
            args.value,
            backup_limit=args.backup_limit,
            dry_run=args.dry_run,
        )
    elif command == "migrate":
        migrate_legacy_keys(dry_run=args.dry_run)
    elif command == "backup":
        manage_backups(args.action, backup_limit=args.backup_limit, dry_run=args.dry_run)
    elif command == "install":
        install()
    else:
        show_help()


if __name__ == "__main__":
    try:
        main()
    except (ClaudesError, OSError) as exc:
        print(f"error: {exc}", file=sys.stderr)
        raise SystemExit(1)
