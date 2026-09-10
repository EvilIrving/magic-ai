#!/usr/bin/env python3
import concurrent.futures
import json
import os
import shlex
import sys
import threading
import time
import urllib.error
import urllib.request
from datetime import datetime, timezone
from pathlib import Path


BASE_URL = "https://api.shu.cool"
HERE = Path(__file__).resolve().parent
MODELS_PATH = HERE / "models.json"
OUTPUT_PATH = HERE / "gateway-capability-matrix.json"
PROFILE_PATHS = (Path.home() / ".zshrc", Path.home() / ".bash_profile", Path.home() / ".bashrc")
PROVIDER_KEY_NAMES = {
    "s-openai": "SHU_API_KEY",
    "s-deepseek": "SHU_DEEPSEEK_API_KEY",
    "s-grok": "SHU_GROK_API_KEY",
}
EFFORTS = ("none", "minimal", "low", "medium", "high", "xhigh", "max")
WRITE_LOCK = threading.Lock()


def load_key(provider_id):
    key_name = PROVIDER_KEY_NAMES[provider_id]
    value = os.environ.get(key_name, "").strip()
    if value:
        return key_name, value, "environment"
    prefix = f"{key_name}="
    for path in PROFILE_PATHS:
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
                    return key_name, value, str(path)
    auth_path = Path.home() / ".pi" / "agent" / "auth.json"
    if auth_path.is_file():
        auth = json.loads(auth_path.read_text(encoding="utf-8"))
        value = str(auth.get(provider_id, {}).get("key", "")).strip()
        if value:
            return key_name, value, "pi-auth"
    raise RuntimeError(f"{key_name} 未配置")


def configured_api(provider, model):
    return model.get("api", provider["api"])


def endpoint_for(api):
    if api == "openai-responses":
        return "responses"
    if api == "openai-completions":
        return "chat_completions"
    raise ValueError(f"未知接口类型：{api}")


def base_payload(protocol, model_id, max_value=32):
    if protocol == "responses":
        return {
            "model": model_id,
            "input": "只返回 OK",
            "max_output_tokens": max_value,
            "stream": False,
        }
    if protocol == "chat_completions":
        return {
            "model": model_id,
            "messages": [{"role": "user", "content": "只返回 OK"}],
            "max_tokens": max_value,
            "stream": False,
        }
    if protocol == "anthropic_messages":
        return {
            "model": model_id,
            "messages": [{"role": "user", "content": "只返回 OK"}],
            "max_tokens": max_value,
            "stream": False,
        }
    raise ValueError(protocol)


def endpoint_path(protocol):
    return {
        "responses": "/v1/responses",
        "chat_completions": "/v1/chat/completions",
        "anthropic_messages": "/v1/messages",
    }[protocol]


def case(case_id, category, provider_id, model, protocol, payload, expectation):
    return {
        "case_id": case_id,
        "category": category,
        "provider": provider_id,
        "credential": PROVIDER_KEY_NAMES[provider_id],
        "model": model["id"],
        "configured_api": model["_configured_api"],
        "configured_max_tokens": model.get("maxTokens"),
        "protocol": protocol,
        "request": {
            "method": "POST",
            "url": BASE_URL + endpoint_path(protocol),
            "headers": {
                "Accept": "application/json",
                "Content-Type": "application/json",
                **(
                    {"anthropic-version": "2023-06-01", "authentication": "x-api-key（已脱敏）"}
                    if protocol == "anthropic_messages"
                    else {"authentication": "Bearer（已脱敏）"}
                ),
            },
            "body": payload,
        },
        "expectation": expectation,
    }


def make_cases(catalog):
    cases = []
    for provider_id, provider, model in catalog:
        model_id = model["id"]
        primary = endpoint_for(model["_configured_api"])

        for protocol in ("responses", "chat_completions", "anthropic_messages"):
            cases.append(case(
                f"endpoint:{model_id}:{protocol}",
                "endpoint_compatibility",
                provider_id,
                model,
                protocol,
                base_payload(protocol, model_id),
                "探测该模型是否接受此协议的最小非流式请求",
            ))

        for effort in EFFORTS:
            payload = base_payload(primary, model_id, 16)
            if primary == "responses":
                payload["reasoning"] = {"effort": effort}
            else:
                payload["reasoning_effort"] = effort
            cases.append(case(
                f"effort:{model_id}:{effort}",
                "thinking_effort",
                provider_id,
                model,
                primary,
                payload,
                f"探测思考强度 {effort} 是否被配置协议接受",
            ))

        for field in ("max_output_tokens", "max_tokens", "max_completion_tokens"):
            payload = base_payload(primary, model_id)
            payload.pop("max_output_tokens", None)
            payload.pop("max_tokens", None)
            payload[field] = 32
            cases.append(case(
                f"token_field:{model_id}:{field}",
                "token_field",
                provider_id,
                model,
                primary,
                payload,
                f"探测长度字段 {field} 是否被接受",
            ))

        max_tokens = model.get("maxTokens")
        if max_tokens:
            primary_field = "max_output_tokens" if primary == "responses" else (
                provider.get("compat", {}).get("maxTokensField", "max_tokens")
            )
            for label, value in (("configured_max", max_tokens), ("configured_max_plus_one", max_tokens + 1)):
                payload = base_payload(primary, model_id)
                payload.pop("max_output_tokens", None)
                payload.pop("max_tokens", None)
                payload[primary_field] = value
                cases.append(case(
                    f"token_boundary:{model_id}:{label}",
                    "token_boundary_acceptance",
                    provider_id,
                    model,
                    primary,
                    payload,
                    "仅探测网关是否接受该参数值，不验证实际生成长度",
                ))

        stream_payload = base_payload(primary, model_id, 32)
        stream_payload["stream"] = True
        cases.append(case(
            f"feature:{model_id}:stream",
            "streaming",
            provider_id,
            model,
            primary,
            stream_payload,
            "探测流式响应及原始事件序列",
        ))

        tool_payload = base_payload(primary, model_id, 64)
        tool_payload["input" if primary == "responses" else "messages"] = (
            "调用 ping 工具，参数 value 设为 ok"
            if primary == "responses"
            else [{"role": "user", "content": "调用 ping 工具，参数 value 设为 ok"}]
        )
        tool_payload["tools"] = [{
            "type": "function",
            **(
                {
                    "name": "ping",
                    "description": "探测工具",
                    "parameters": {
                        "type": "object",
                        "properties": {"value": {"type": "string"}},
                        "required": ["value"],
                        "additionalProperties": False,
                    },
                    "strict": True,
                }
                if primary == "responses"
                else {
                    "function": {
                        "name": "ping",
                        "description": "探测工具",
                        "parameters": {
                            "type": "object",
                            "properties": {"value": {"type": "string"}},
                            "required": ["value"],
                            "additionalProperties": False,
                        },
                        "strict": True,
                    }
                }
            ),
        }]
        cases.append(case(
            f"feature:{model_id}:strict_tool",
            "strict_tool_call",
            provider_id,
            model,
            primary,
            tool_payload,
            "探测严格函数工具定义与工具调用",
        ))

        schema = {
            "type": "object",
            "properties": {"result": {"type": "string"}},
            "required": ["result"],
            "additionalProperties": False,
        }
        schema_payload = base_payload(primary, model_id, 64)
        schema_payload["input" if primary == "responses" else "messages"] = (
            "返回 result 为 ok 的对象"
            if primary == "responses"
            else [{"role": "user", "content": "返回 result 为 ok 的对象"}]
        )
        if primary == "responses":
            schema_payload["text"] = {
                "format": {"type": "json_schema", "name": "probe", "strict": True, "schema": schema}
            }
        else:
            schema_payload["response_format"] = {
                "type": "json_schema",
                "json_schema": {"name": "probe", "strict": True, "schema": schema},
            }
        cases.append(case(
            f"feature:{model_id}:structured_output",
            "structured_output",
            provider_id,
            model,
            primary,
            schema_payload,
            "探测严格 JSON Schema 输出",
        ))
    return cases


def parse_sse(raw):
    events = []
    current = {}
    for line in raw.splitlines():
        if not line:
            if current:
                events.append(current)
                current = {}
            continue
        if line.startswith("event:"):
            current["event"] = line[6:].strip()
        elif line.startswith("data:"):
            text = line[5:].strip()
            try:
                data = json.loads(text)
            except json.JSONDecodeError:
                data = text
            current.setdefault("data", []).append(data)
        elif line.startswith("id:"):
            current["id"] = line[3:].strip()
        else:
            current.setdefault("other_lines", []).append(line)
    if current:
        events.append(current)
    return events


def execute(item):
    key_name, key, key_source = load_key(item["provider"])
    protocol = item["protocol"]
    headers = {
        "Accept": "application/json",
        "Content-Type": "application/json",
        "User-Agent": "gateway-capability-probe/1.0",
    }
    if protocol == "anthropic_messages":
        headers["x-api-key"] = key
        headers["anthropic-version"] = "2023-06-01"
    else:
        headers["Authorization"] = f"Bearer {key}"
    data = json.dumps(item["request"]["body"], ensure_ascii=False).encode()
    request = urllib.request.Request(item["request"]["url"], data=data, headers=headers, method="POST")
    started = time.monotonic()
    try:
        response = urllib.request.urlopen(request, timeout=240)
    except urllib.error.HTTPError as error:
        response = error
    raw = response.read().decode("utf-8", errors="replace")
    content_type = response.headers.get("content-type", "")
    body = None
    events = None
    if "text/event-stream" in content_type or raw.startswith(("event:", "data:")):
        events = parse_sse(raw)
    else:
        try:
            body = json.loads(raw)
        except json.JSONDecodeError:
            pass
    item["credential_source"] = key_source
    item["response"] = {
        "status": response.status,
        "classification": "accepted" if response.status < 400 else "rejected",
        "headers": {
            name.lower(): value
            for name, value in response.headers.items()
            if name.lower() in {
                "content-type", "x-request-id", "request-id", "openai-request-id",
                "anthropic-ratelimit-requests-limit", "anthropic-ratelimit-tokens-limit",
            }
        },
        "elapsed_ms": round((time.monotonic() - started) * 1000),
        "body": body,
        "sse_events": events,
        "raw_text": raw,
    }
    return item


def write_report(report):
    with WRITE_LOCK:
        OUTPUT_PATH.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def confirm_existing_rejections():
    report = json.loads(OUTPUT_PATH.read_text(encoding="utf-8"))
    rejected = [
        item for item in report["results"]
        if item.get("response", {}).get("classification") == "rejected"
    ]
    print(f"复测 {len(rejected)} 条拒绝结果，每条两次", file=sys.stderr, flush=True)
    attempts = [(item, attempt) for item in rejected for attempt in (1, 2)]
    with concurrent.futures.ThreadPoolExecutor(max_workers=4) as executor:
        futures = {executor.submit(execute, dict(item)): (item, attempt) for item, attempt in attempts}
        for index, future in enumerate(concurrent.futures.as_completed(futures), 1):
            original, attempt = futures[future]
            try:
                result = future.result()
                response = result["response"]
            except Exception as error:
                response = {
                    "classification": "client_error",
                    "error_type": type(error).__name__,
                    "message": str(error),
                }
            original.setdefault("confirmation_attempts", []).append({
                "attempt": attempt,
                "response": response,
            })
            if index % 10 == 0 or index == len(attempts):
                print(f"[{index}/{len(attempts)}]", file=sys.stderr, flush=True)
                write_report(report)
    for item in rejected:
        classes = [
            attempt["response"]["classification"]
            for attempt in item.get("confirmation_attempts", [])
        ]
        item["confirmed_classification"] = (
            "flaky" if "accepted" in classes
            else "consistently_rejected" if classes and set(classes) == {"rejected"}
            else "client_error"
        )
    report["summary"]["rejection_confirmation"] = {
        "tested": len(rejected),
        "consistently_rejected": sum(
            item.get("confirmed_classification") == "consistently_rejected" for item in rejected
        ),
        "flaky": sum(item.get("confirmed_classification") == "flaky" for item in rejected),
        "client_errors": sum(item.get("confirmed_classification") == "client_error" for item in rejected),
    }
    write_report(report)
    print(OUTPUT_PATH)
    return 0


def main():
    if len(sys.argv) == 2 and sys.argv[1] == "--confirm-existing-rejections":
        return confirm_existing_rejections()
    source = json.loads(MODELS_PATH.read_text(encoding="utf-8"))
    catalog = []
    model_summary = []
    for provider_id, provider in source["providers"].items():
        for original in provider["models"]:
            model = dict(original)
            model["_configured_api"] = configured_api(provider, model)
            catalog.append((provider_id, provider, model))
            model_summary.append({
                "provider": provider_id,
                "model": model["id"],
                "configured_api": model["_configured_api"],
                "configured_max_tokens": model.get("maxTokens"),
                "configured_context_window": model.get("contextWindow"),
                "configured_thinking_level_map": model.get("thinkingLevelMap"),
            })
    cases = make_cases(catalog)
    report = {
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "gateway": BASE_URL,
        "secrets_included": False,
        "scope": {
            "model_count": len(catalog),
            "case_count": len(cases),
            "notes": [
                "长度边界用短输出探测参数是否被接受，不等同于验证实际最大生成长度。",
                "上下文窗口未注入等量令牌，配置值仅作为对照保留。",
                "HTTP 2xx 表示请求被接受，具体能力是否正确还需结合响应体判断。",
            ],
        },
        "models": model_summary,
        "summary": {
            "completed": 0,
            "accepted": 0,
            "rejected": 0,
            "client_errors": 0,
        },
        "results": [],
    }
    write_report(report)
    print(f"共 {len(cases)} 项，使用 4 路并发", file=sys.stderr, flush=True)
    with concurrent.futures.ThreadPoolExecutor(max_workers=4) as executor:
        futures = {executor.submit(execute, item): item for item in cases}
        for index, future in enumerate(concurrent.futures.as_completed(futures), 1):
            original = futures[future]
            try:
                result = future.result()
            except Exception as error:
                result = original
                result["response"] = {
                    "classification": "client_error",
                    "error_type": type(error).__name__,
                    "message": str(error),
                }
            report["results"].append(result)
            classification = result["response"]["classification"]
            report["summary"]["completed"] += 1
            if classification == "accepted":
                report["summary"]["accepted"] += 1
            elif classification == "rejected":
                report["summary"]["rejected"] += 1
            else:
                report["summary"]["client_errors"] += 1
            if index % 10 == 0 or index == len(cases):
                print(
                    f"[{index}/{len(cases)}] 接受 {report['summary']['accepted']}，"
                    f"拒绝 {report['summary']['rejected']}，异常 {report['summary']['client_errors']}",
                    file=sys.stderr,
                    flush=True,
                )
                write_report(report)
    report["results"].sort(key=lambda item: item["case_id"])
    report["generated_at"] = datetime.now(timezone.utc).isoformat()
    write_report(report)
    print(OUTPUT_PATH)
    return 0 if report["summary"]["client_errors"] == 0 else 1


if __name__ == "__main__":
    raise SystemExit(main())
