#!/usr/bin/env python3
import json
import os
import shlex
import sys
import time
import urllib.error
import urllib.request
from datetime import datetime, timezone
from pathlib import Path


BASE_URL = "https://api.shu.cool"
OUTPUT_PATH = Path(__file__).with_name("gateway-raw-responses.json")
PROFILES = (Path.home() / ".zshrc", Path.home() / ".bash_profile", Path.home() / ".bashrc")
KEY_NAMES = ("SHU_API_KEY", "SHU_DEEPSEEK_API_KEY", "SHU_GROK_API_KEY")
PI_PROVIDER_BY_KEY = {
    "SHU_API_KEY": "s-openai",
    "SHU_DEEPSEEK_API_KEY": "s-deepseek",
    "SHU_GROK_API_KEY": "s-grok",
}


def load_key(name):
    value = os.environ.get(name, "").strip()
    if value:
        return value
    prefix = f"{name}="
    for path in PROFILES:
        if not path.is_file():
            continue
        for line in reversed(path.read_text(encoding="utf-8", errors="replace").splitlines()):
            try:
                tokens = shlex.split(line, comments=True, posix=True)
            except ValueError:
                continue
            if len(tokens) == 2 and tokens[0] == "export" and tokens[1].startswith(prefix):
                value = tokens[1][len(prefix):].strip()
                if value:
                    return value
    auth_path = Path.home() / ".pi" / "agent" / "auth.json"
    if auth_path.is_file():
        auth = json.loads(auth_path.read_text(encoding="utf-8"))
        credential = auth.get(PI_PROVIDER_BY_KEY[name], {})
        value = credential.get("key", "").strip()
        if value:
            return value
    raise RuntimeError(f"{name} 未配置")


def call(name, key_name, method, path, payload=None, anthropic=False):
    key = load_key(key_name)
    headers = {"Accept": "application/json", "User-Agent": "gateway-format-probe/1.0"}
    if anthropic:
        headers["x-api-key"] = key
        headers["anthropic-version"] = "2023-06-01"
    else:
        headers["Authorization"] = f"Bearer {key}"
    data = None
    if payload is not None:
        headers["Content-Type"] = "application/json"
        data = json.dumps(payload, ensure_ascii=False).encode()
    request = urllib.request.Request(BASE_URL + path, data=data, headers=headers, method=method)
    started = time.monotonic()
    try:
        response = urllib.request.urlopen(request, timeout=180)
    except urllib.error.HTTPError as error:
        response = error
    raw = response.read().decode("utf-8", errors="replace")
    try:
        body = json.loads(raw)
    except json.JSONDecodeError:
        body = None
    response_headers = {
        key.lower(): value
        for key, value in response.headers.items()
        if key.lower() in {"content-type", "x-request-id", "request-id", "openai-request-id"}
    }
    return {
        "name": name,
        "request": {
            "method": method,
            "url": BASE_URL + path,
            "credential": key_name,
            "headers": {
                key: value for key, value in headers.items()
                if key.lower() not in {"authorization", "x-api-key"}
            },
            "body": payload,
        },
        "response": {
            "status": response.status,
            "headers": response_headers,
            "elapsed_ms": round((time.monotonic() - started) * 1000),
            "body": body,
            "raw_text": raw,
        },
    }


def main():
    probes = []
    errors = []
    groups = {
        "openai": ("SHU_API_KEY", "gpt-5.6-sol"),
        "deepseek": ("SHU_DEEPSEEK_API_KEY", "deepseek-v4-flash"),
        "grok": ("SHU_GROK_API_KEY", "grok-4.5"),
    }
    response_models = {
        "SHU_API_KEY": [
            "gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna", "gpt-5.5",
            "gpt-5.4", "gpt-5.4-mini", "gpt-5.3-codex-spark",
        ],
        "SHU_GROK_API_KEY": ["grok-4.5"],
    }
    chat_models = {
        "SHU_DEEPSEEK_API_KEY": ["deepseek-v4-flash", "deepseek-v4-pro"],
        "SHU_GROK_API_KEY": ["grok-4.3", "grok-build", "grok-composer-2.5-fast"],
    }

    requests = []
    for group, (key_name, _) in groups.items():
        requests.append((f"models:{group}", key_name, "GET", "/v1/models", None, False))
    for key_name, models in response_models.items():
        for model in models:
            payload = {"model": model, "input": "只返回 OK", "max_output_tokens": 32, "stream": False}
            requests.append((f"responses:{model}", key_name, "POST", "/v1/responses", payload, False))
    for key_name, models in chat_models.items():
        for model in models:
            payload = {
                "model": model,
                "messages": [{"role": "user", "content": "只返回 OK"}],
                "max_tokens": 32,
                "stream": False,
            }
            requests.append((f"chat_completions:{model}", key_name, "POST", "/v1/chat/completions", payload, False))
    for group, (key_name, model) in groups.items():
        payload = {
            "model": model,
            "max_tokens": 32,
            "messages": [{"role": "user", "content": "只返回 OK"}],
        }
        requests.append((f"anthropic_messages:{group}:{model}", key_name, "POST", "/v1/messages", payload, True))

    for index, args in enumerate(requests, 1):
        print(f"[{index}/{len(requests)}] {args[0]}", file=sys.stderr, flush=True)
        try:
            probes.append(call(*args))
        except Exception as error:
            errors.append({"name": args[0], "error_type": type(error).__name__, "message": str(error)})

    report = {
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "gateway": BASE_URL,
        "scope": "当前 providers/pi/models.json 与 providers/claudes/claudes.sh 配置的非流式接口组合",
        "secrets_included": False,
        "summary": {
            "probe_count": len(probes),
            "success_count": sum(item["response"]["status"] < 400 for item in probes),
            "http_error_count": sum(item["response"]["status"] >= 400 for item in probes),
            "client_error_count": len(errors),
        },
        "probes": probes,
        "client_errors": errors,
    }
    OUTPUT_PATH.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(OUTPUT_PATH)
    return 1 if errors else 0


if __name__ == "__main__":
    raise SystemExit(main())
