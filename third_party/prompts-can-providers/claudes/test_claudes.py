from __future__ import annotations

import json
import os
import re
import runpy
import stat
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


SCRIPT = Path(__file__).with_name("claudes.sh")
WINDOWS_SCRIPT = Path(__file__).with_name("claudes.ps1")
WINDOWS_STATUSLINE = Path(__file__).with_name("statusline.ps1")


class ClaudesCliTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temp_dir = tempfile.TemporaryDirectory()
        self.home = Path(self.temp_dir.name)
        self.claude_home = self.home / ".claude"
        self.env = {
            **os.environ,
            "HOME": str(self.home),
            "CLAUDE_HOME": str(self.claude_home),
            "SHELL": "/bin/zsh",
        }
        for key in (
            "DEEPSEEK_API_KEY",
            "KIMI_API_KEY",
            "MINIMAX_API_KEY",
            "SHU_API_KEY",
            "SHU_DEEPSEEK_API_KEY",
            "SHU_GROK_API_KEY",
        ):
            self.env.pop(key, None)

    def tearDown(self) -> None:
        self.temp_dir.cleanup()

    def run_cli(self, *arguments: str, check: bool = True) -> subprocess.CompletedProcess[str]:
        result = subprocess.run(
            [sys.executable, str(SCRIPT), *arguments],
            env=self.env,
            text=True,
            capture_output=True,
            check=False,
        )
        if check and result.returncode != 0:
            self.fail(f"command failed: {result.stderr}\n{result.stdout}")
        return result

    def write_json(self, path: Path, value: object, mode: int = 0o644) -> None:
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(json.dumps(value), encoding="utf-8")
        path.chmod(mode)

    def read_settings(self) -> dict[str, object]:
        return json.loads((self.claude_home / "settings.json").read_text(encoding="utf-8"))

    def configure_legacy_keys(self) -> None:
        self.write_json(
            self.claude_home / "models.json",
            {
                "providers": {
                    "deepseek": {"apiKey": "deepseek-secret"},
                    "kimi": {"apiKey": "kimi-secret"},
                    "minimax": {"apiKey": "minimax-secret"},
                    "s-openai": {"apiKey": "shu-openai-secret"},
                    "s-deepseek": {"apiKey": "shu-deepseek-secret"},
                    "s-grok": {"apiKey": "shu-grok-secret"},
                }
            },
            0o600,
        )
        self.run_cli("migrate")

    def test_official_preserves_unmanaged_settings_and_removes_provider_fields(self) -> None:
        settings_path = self.claude_home / "settings.json"
        self.write_json(
            settings_path,
            {
                "permissions": {"allow": ["Read(*)"]},
                "env": {
                    "CUSTOM_SETTING": "keep",
                    "ANTHROPIC_BASE_URL": "https://old.example",
                    "ANTHROPIC_AUTH_TOKEN": "old-secret",
                    "ANTHROPIC_MODEL": "old-model",
                },
            },
        )

        self.run_cli("use", "official")

        settings = self.read_settings()
        self.assertEqual(settings["permissions"], {"allow": ["Read(*)"]})
        self.assertEqual(settings["env"], {"CUSTOM_SETTING": "keep"})
        self.assertEqual(stat.S_IMODE(settings_path.stat().st_mode), 0o600)
        self.assertEqual(len(list((self.claude_home / "provider-switch-backups").iterdir())), 1)

    def test_legacy_keys_migrate_and_switch_between_official_apis(self) -> None:
        self.configure_legacy_keys()
        profile = (self.home / ".zshrc").read_text(encoding="utf-8")
        self.assertIn("DEEPSEEK_API_KEY=", profile)
        self.assertIn("KIMI_API_KEY=", profile)
        self.assertIn("MINIMAX_API_KEY=", profile)
        self.assertIn("SHU_API_KEY=", profile)
        self.assertIn("SHU_DEEPSEEK_API_KEY=", profile)
        self.assertIn("SHU_GROK_API_KEY=", profile)

        self.run_cli("use", "deepseek")
        deepseek = self.read_settings()["env"]
        self.assertEqual(deepseek["ANTHROPIC_BASE_URL"], "https://api.deepseek.com/anthropic")
        self.assertEqual(deepseek["ANTHROPIC_AUTH_TOKEN"], "deepseek-secret")
        self.assertEqual(deepseek["ANTHROPIC_MODEL"], "deepseek-v4-pro")

        self.run_cli("use", "kimi")
        kimi = self.read_settings()["env"]
        self.assertEqual(kimi["ANTHROPIC_AUTH_TOKEN"], "kimi-secret")
        self.assertEqual(kimi["ANTHROPIC_REASONING_MODEL"], "kimi-k2.5")

        self.run_cli("use", "minimax")
        minimax = self.read_settings()["env"]
        self.assertEqual(minimax["ANTHROPIC_AUTH_TOKEN"], "minimax-secret")
        self.assertEqual(minimax["ANTHROPIC_MODEL"], "MiniMax-M2.7")

    def test_shu_routes_with_separate_keys_are_detected_by_model(self) -> None:
        for provider, secret in (
            ("s-openai", "shu-openai-secret"),
            ("s-deepseek", "shu-deepseek-secret"),
            ("s-grok", "shu-grok-secret"),
        ):
            self.run_cli("env", provider, secret)
        for provider, model in (
            ("s-openai", "gpt-5.6-sol"),
            ("s-deepseek", "deepseek-v4-pro"),
            ("s-grok", "grok-4.5"),
        ):
            self.run_cli("use", provider)
            settings = self.read_settings()
            self.assertEqual(settings["env"]["ANTHROPIC_MODEL"], model)
            status = self.run_cli("status").stdout
            self.assertIn(f"Provider      : {provider}", status)
        settings = self.read_settings()
        self.assertEqual(settings["env"]["ANTHROPIC_DEFAULT_SONNET_MODEL"], "grok-4.5")
        self.assertNotEqual(settings["env"]["ANTHROPIC_DEFAULT_SONNET_MODEL"], "grok-build")

    def test_backup_restore_returns_to_previous_provider(self) -> None:
        self.configure_legacy_keys()
        self.run_cli("use", "deepseek")
        self.run_cli("use", "kimi")

        self.run_cli("backup", "restore")

        settings = self.read_settings()
        self.assertEqual(settings["env"]["ANTHROPIC_MODEL"], "deepseek-v4-pro")

    def test_install_copies_only_the_script_and_updates_alias(self) -> None:
        self.run_cli("install")

        target = self.claude_home / "claudes"
        self.assertTrue(target.is_file())
        self.assertEqual(stat.S_IMODE(target.stat().st_mode), 0o755)
        self.assertFalse((self.claude_home / "models.json").exists())
        profile = (self.home / ".zshrc").read_text(encoding="utf-8")
        self.assertIn(f'alias claudes="{target}"', profile)

    def test_missing_key_does_not_change_settings(self) -> None:
        result = self.run_cli("use", "deepseek", check=False)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("DEEPSEEK_API_KEY is not set", result.stderr)
        self.assertFalse((self.claude_home / "settings.json").exists())

    def test_reusing_active_provider_repairs_settings_permissions(self) -> None:
        self.run_cli("env", "shu", "shu-secret")
        self.run_cli("use", "s-openai")
        settings_path = self.claude_home / "settings.json"
        settings_path.chmod(0o644)

        result = self.run_cli("use", "s-openai")

        self.assertIn("Permissions fixed", result.stdout)
        self.assertEqual(stat.S_IMODE(settings_path.stat().st_mode), 0o600)

    def test_clearing_active_key_removes_embedded_credential(self) -> None:
        self.run_cli("env", "deepseek", "deepseek-secret")
        self.run_cli("use", "deepseek")

        self.run_cli("env", "deepseek", "clear")

        settings = self.read_settings()
        self.assertNotIn("ANTHROPIC_AUTH_TOKEN", settings["env"])
        self.assertEqual(settings["env"]["ANTHROPIC_MODEL"], "deepseek-v4-pro")
        self.assertEqual(len(list((self.claude_home / "provider-switch-backups").iterdir())), 1)

    def test_windows_catalog_matches_unix_catalog(self) -> None:
        namespace = runpy.run_path(str(SCRIPT))
        unix_catalog = {
            provider_id: {
                "name": provider.name,
                "baseUrl": provider.base_url,
                "keyEnv": provider.key_env,
                "models": provider.models,
            }
            for provider_id, provider in namespace["PROVIDERS"].items()
        }
        windows_source = WINDOWS_SCRIPT.read_text(encoding="utf-8")
        match = re.search(
            r"\$script:ProviderCatalogJson = @'\n(.*?)\n'@",
            windows_source,
            re.DOTALL,
        )

        self.assertIsNotNone(match)
        windows_catalog = json.loads(match.group(1))
        self.assertEqual(windows_catalog, unix_catalog)

    def test_windows_installer_defines_the_command_wrapper(self) -> None:
        source = WINDOWS_SCRIPT.read_text(encoding="utf-8")

        self.assertIn("Join-Path $ClaudeHome 'claudes.ps1'", source)
        self.assertIn("Join-Path $ClaudeHome 'claudes.cmd'", source)
        self.assertIn('ExecutionPolicy Bypass -File "%~dp0claudes.ps1" %*', source)
        self.assertIn("Join-Path $sourceDir 'statusline.ps1'", source)
        self.assertIn("Join-Path $ClaudeHome 'statusline.ps1'", source)
        self.assertIn("$updated['statusLine'] = [ordered]@{", source)
        self.assertTrue(WINDOWS_STATUSLINE.is_file())


if __name__ == "__main__":
    unittest.main()
