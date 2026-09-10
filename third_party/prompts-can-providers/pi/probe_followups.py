#!/usr/bin/env python3
import json
import urllib.error
import urllib.request
from datetime import datetime, timezone
from pathlib import Path

import probe_capabilities as core


OUTPUT_PATH = Path(__file__).with_name("gateway-followup-raw-responses.json")


def model_detail(provider_id, model_id):
    key_name, key, key_source = core.load_key(provider_id)
    url = f"{core.BASE_URL}/v1/models/{model_id}"
    request = urllib.request.Request(
        url,
        headers={"Authorization": f"Bearer {key}", "Accept": "application/json"},
    )
    try:
        response = urllib.request.urlopen(request, timeout=30)
    except urllib.error.HTTPError as error:
        response = error
    raw = response.read().decode("utf-8", errors="replace")
    try:
        body = json.loads(raw)
    except json.JSONDecodeError:
        body = None
    return {
        "case_id": f"model_detail:{provider_id}:{model_id}",
        "credential": key_name,
        "credential_source": key_source,
        "request": {
            "method": "GET",
            "url": url,
            "headers": {
                "Accept": "application/json",
                "authentication": "Bearer（已脱敏）",
            },
            "body": None,
        },
        "response": {
            "status": response.status,
            "headers": {
                name.lower(): value
                for name, value in response.headers.items()
                if name.lower() in {"content-type", "x-request-id", "request-id"}
            },
            "body": body,
            "raw_text": raw,
        },
    }


def spark_structured_output():
    source = json.loads(core.MODELS_PATH.read_text(encoding="utf-8"))
    provider = source["providers"]["s-openai"]
    model = dict(next(item for item in provider["models"] if item["id"] == "gpt-5.3-codex-spark"))
    model["_configured_api"] = core.configured_api(provider, model)
    schema = {
        "type": "object",
        "properties": {"result": {"type": "string"}},
        "required": ["result"],
        "additionalProperties": False,
    }
    payload = core.base_payload("responses", model["id"], 256)
    payload["input"] = "返回 result 为 ok 的对象"
    payload["text"] = {
        "format": {
            "type": "json_schema",
            "name": "probe",
            "strict": True,
            "schema": schema,
        }
    }
    item = core.case(
        "followup:gpt-5.3-codex-spark:structured_output_256",
        "structured_output_followup",
        "s-openai",
        model,
        "responses",
        payload,
        "提高输出上限后复测结构化输出",
    )
    return core.execute(item)


def main():
    report = {
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "gateway": core.BASE_URL,
        "secrets_included": False,
        "results": [
            model_detail("s-openai", "gpt-5.6-sol"),
            model_detail("s-deepseek", "deepseek-v4-flash"),
            model_detail("s-grok", "grok-4.5"),
            spark_structured_output(),
        ],
    }
    OUTPUT_PATH.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(OUTPUT_PATH)


if __name__ == "__main__":
    main()
