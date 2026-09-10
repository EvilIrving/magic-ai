#!/usr/bin/env python3
import concurrent.futures
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

import probe_capabilities as core


OUTPUT_PATH = Path(__file__).with_name("gateway-chat-completions-matrix.json")


def make_case(case_id, category, provider_id, model, payload, expectation):
    return core.case(
        case_id,
        category,
        provider_id,
        model,
        "chat_completions",
        payload,
        expectation,
    )


def base(model_id, max_tokens=32):
    return {
        "model": model_id,
        "messages": [{"role": "user", "content": "只返回 OK"}],
        "max_tokens": max_tokens,
        "stream": False,
    }


def build_cases(catalog):
    cases = []
    for provider_id, provider, model in catalog:
        model_id = model["id"]
        for effort in core.EFFORTS:
            payload = base(model_id, 16)
            payload["reasoning_effort"] = effort
            cases.append(make_case(
                f"chat_effort:{model_id}:{effort}",
                "thinking_effort",
                provider_id,
                model,
                payload,
                f"探测聊天补全协议的思考强度 {effort}",
            ))

        for field in ("max_tokens", "max_completion_tokens"):
            payload = base(model_id)
            payload.pop("max_tokens")
            payload[field] = 32
            cases.append(make_case(
                f"chat_token_field:{model_id}:{field}",
                "token_field",
                provider_id,
                model,
                payload,
                f"探测聊天补全协议的长度字段 {field}",
            ))

        configured_max = model.get("maxTokens")
        if configured_max:
            for field in ("max_tokens", "max_completion_tokens"):
                for label, value in (
                    ("configured_max", configured_max),
                    ("configured_max_plus_one", configured_max + 1),
                ):
                    payload = base(model_id)
                    payload.pop("max_tokens")
                    payload[field] = value
                    cases.append(make_case(
                        f"chat_token_boundary:{model_id}:{field}:{label}",
                        "token_boundary_acceptance",
                        provider_id,
                        model,
                        payload,
                        "仅探测聊天补全网关是否接受该参数值，不验证实际生成长度",
                    ))

        stream_payload = base(model_id)
        stream_payload["stream"] = True
        stream_payload["stream_options"] = {"include_usage": True}
        cases.append(make_case(
            f"chat_feature:{model_id}:stream",
            "streaming",
            provider_id,
            model,
            stream_payload,
            "探测聊天补全流式事件和用量尾包",
        ))

        developer_payload = base(model_id)
        developer_payload["messages"] = [
            {"role": "developer", "content": "只允许回答 OK"},
            {"role": "user", "content": "执行要求"},
        ]
        cases.append(make_case(
            f"chat_feature:{model_id}:developer_role",
            "developer_role",
            provider_id,
            model,
            developer_payload,
            "探测聊天补全开发者角色",
        ))

        tool_payload = base(model_id, 64)
        tool_payload["messages"] = [
            {"role": "user", "content": "调用 ping 工具，参数 value 设为 ok"}
        ]
        tool_payload["tools"] = [{
            "type": "function",
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
            },
        }]
        cases.append(make_case(
            f"chat_feature:{model_id}:strict_tool",
            "strict_tool_call",
            provider_id,
            model,
            tool_payload,
            "探测聊天补全严格函数工具定义与工具调用",
        ))

        schema = {
            "type": "object",
            "properties": {"result": {"type": "string"}},
            "required": ["result"],
            "additionalProperties": False,
        }
        schema_payload = base(model_id, 64)
        schema_payload["messages"] = [
            {"role": "user", "content": "返回 result 为 ok 的对象"}
        ]
        schema_payload["response_format"] = {
            "type": "json_schema",
            "json_schema": {"name": "probe", "strict": True, "schema": schema},
        }
        cases.append(make_case(
            f"chat_feature:{model_id}:structured_output",
            "structured_output",
            provider_id,
            model,
            schema_payload,
            "探测聊天补全严格 JSON Schema 输出",
        ))
    return cases


def confirm_rejections():
    report = json.loads(OUTPUT_PATH.read_text(encoding="utf-8"))
    rejected = [
        item for item in report["results"]
        if item.get("response", {}).get("classification") == "rejected"
    ]
    attempts = [(item, attempt) for item in rejected for attempt in (1, 2)]
    print(f"复测 {len(rejected)} 条拒绝结果，每条两次", file=sys.stderr, flush=True)
    with concurrent.futures.ThreadPoolExecutor(max_workers=4) as executor:
        futures = {executor.submit(core.execute, dict(item)): (item, attempt) for item, attempt in attempts}
        for index, future in enumerate(concurrent.futures.as_completed(futures), 1):
            original, attempt = futures[future]
            try:
                response = future.result()["response"]
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
                OUTPUT_PATH.write_text(
                    json.dumps(report, ensure_ascii=False, indent=2) + "\n",
                    encoding="utf-8",
                )
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
    OUTPUT_PATH.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(OUTPUT_PATH)
    return 0


def main():
    if len(sys.argv) == 2 and sys.argv[1] == "--confirm-rejections":
        return confirm_rejections()
    source = json.loads(core.MODELS_PATH.read_text(encoding="utf-8"))
    catalog = []
    models = []
    for provider_id, provider in source["providers"].items():
        for original in provider["models"]:
            model = dict(original)
            model["_configured_api"] = core.configured_api(provider, model)
            catalog.append((provider_id, provider, model))
            models.append({
                "provider": provider_id,
                "model": model["id"],
                "configured_api": model["_configured_api"],
                "configured_max_tokens": model.get("maxTokens"),
                "configured_context_window": model.get("contextWindow"),
            })
    cases = build_cases(catalog)
    report = {
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "gateway": core.BASE_URL,
        "protocol": "chat_completions",
        "secrets_included": False,
        "scope": {
            "model_count": len(catalog),
            "case_count": len(cases),
            "notes": [
                "长度边界仅探测参数接受情况，不验证实际最大生成长度。",
                "成功状态还需结合工具调用、结构化输出及流式事件语义判断。",
            ],
        },
        "models": models,
        "summary": {"completed": 0, "accepted": 0, "rejected": 0, "client_errors": 0},
        "results": [],
    }
    OUTPUT_PATH.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"共 {len(cases)} 项，使用 4 路并发", file=sys.stderr, flush=True)
    with concurrent.futures.ThreadPoolExecutor(max_workers=4) as executor:
        futures = {executor.submit(core.execute, item): item for item in cases}
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
            if classification in report["summary"]:
                report["summary"][classification] += 1
            else:
                report["summary"]["client_errors"] += 1
            if index % 10 == 0 or index == len(cases):
                print(
                    f"[{index}/{len(cases)}] 接受 {report['summary']['accepted']}，"
                    f"拒绝 {report['summary']['rejected']}，异常 {report['summary']['client_errors']}",
                    file=sys.stderr,
                    flush=True,
                )
                OUTPUT_PATH.write_text(
                    json.dumps(report, ensure_ascii=False, indent=2) + "\n",
                    encoding="utf-8",
                )
    report["results"].sort(key=lambda item: item["case_id"])
    OUTPUT_PATH.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(OUTPUT_PATH)
    return 0 if report["summary"]["client_errors"] == 0 else 1


if __name__ == "__main__":
    raise SystemExit(main())
