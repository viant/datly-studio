#!/usr/bin/env python3
"""Check a real Claude/MCP authoring run; a preamble or release alone cannot pass."""
import argparse
import json
import pathlib
import sys


def decode(value):
    if isinstance(value, list):
        value = "\n".join(item.get("text", "") for item in value if isinstance(item, dict))
    if isinstance(value, str):
        try:
            return json.loads(value)
        except json.JSONDecodeError:
            return value
    return value


def check(path, existing_report):
    calls, results, failures, final = [], {}, [], None
    for line in pathlib.Path(path).read_text().splitlines():
        try:
            message = json.loads(line)
        except json.JSONDecodeError:
            continue
        for block in message.get("message", {}).get("content", []):
            if not isinstance(block, dict):
                continue
            if block.get("type") == "tool_use":
                calls.append(block)
            elif block.get("type") == "tool_result":
                result = decode(block.get("content"))
                results[block.get("tool_use_id")] = result
                if block.get("is_error"):
                    failures.append({"toolUseId": block.get("tool_use_id"), "error": str(result)[:300]})
        if message.get("type") == "result":
            final = message
    errors = ["tool error: " + str(item) for item in failures]
    if not final or final.get("is_error"):
        errors.append("successful final response is missing")
    stages = ["skills_list", "skills_get", "components_create", "versions_create", "versions_apply", "versions_inspect", "versions_validate", "publications_publish", "runtime_status", "catalog_tools_list", "catalog_tools_call"]
    for stage in stages:
        if not any(call["name"].endswith("__studio_sdk_" + stage) or call["name"].endswith("__" + stage) for call in calls):
            errors.append("missing stage: " + stage)
    created = [call for call in calls if call["name"].endswith("__studio_sdk_components_create")]
    report_id = None
    if len(created) != 1:
        errors.append("expected exactly one component creation")
    else:
        result = results.get(created[0]["id"])
        report_id = result.get("id") if isinstance(result, dict) else None
        if not report_id or report_id == existing_report:
            errors.append("new component identity is missing or reused")
    for call in calls:
        if call["name"].endswith(tuple("__studio_sdk_" + name for name in ["versions_apply", "versions_builder", "publications_publish", "resources_upsert_file"])):
            if call.get("input", {}).get("reportId") != report_id:
                errors.append("mutation targeted another component")
    edits = [call for call in calls if call["name"].endswith("__studio_sdk_versions_apply")]
    if len(edits) != 1:
        errors.append("inline first-pass fixture requires one complete DQL apply, without corrective edits")
    validated = [call for call in calls if call["name"].endswith("__studio_sdk_versions_validate")]
    revision = None
    if validated:
        result = results.get(validated[-1]["id"])
        if not isinstance(result, dict) or result.get("valid") is not True:
            errors.append("exact revision validation did not pass")
        revision = validated[-1].get("input", {}).get("expectedSourceRevision")
    published = [call for call in calls if call["name"].endswith("__studio_sdk_publications_publish")]
    publication = results.get(published[-1]["id"]) if published else None
    if not isinstance(publication, dict) or publication.get("status") != "active" or not publication.get("activeGeneration"):
        errors.append("publication did not confirm an active runtime generation")
    if published and published[-1].get("input", {}).get("input", {}).get("expectedSourceRevision") != revision:
        errors.append("published revision differs from validated revision")
    advertised = set()
    for call in calls:
        if call["name"].endswith("__studio_sdk_catalog_tools_list"):
            result = results.get(call["id"])
            if isinstance(result, dict):
                advertised.update(tool.get("name") for tool in result.get("tools", []))
    invocations = [call for call in calls if call["name"].endswith("__studio_sdk_catalog_tools_call")]
    if len(invocations) < 2:
        errors.append("full-key and reduced-grouping MCP calls are required")
    for call in invocations:
        result = results.get(call["id"])
        nested = result.get("result") if isinstance(result, dict) else None
        if not isinstance(nested, dict) or nested.get("isError"):
            errors.append("namespace MCP invocation failed")
        if call.get("input", {}).get("name") not in advertised:
            errors.append("invoked tool was not advertised by the live catalog")
        arguments = call.get("input", {}).get("arguments", {})
        if arguments.get("limit", 0) not in range(1, 11):
            errors.append("MCP call must explicitly limit output to 1-10 rows")
        if "2026-10-01" not in json.dumps(arguments) or "108508" not in json.dumps(arguments):
            errors.append("MCP call lacks the authorized date/advertiser scope")
    def row_sets(value):
        value = decode(value)
        if not isinstance(value, dict):
            return []
        found = []
        for key, child in value.items():
            if key.lower() in ("rows", "data") and isinstance(child, list):
                found.append(child)
            elif isinstance(child, dict):
                found.extend(row_sets(child))
        return found
    for call in invocations:
        result = results.get(call["id"], {})
        nested = result.get("result", {}) if isinstance(result, dict) else {}
        payload = nested.get("structuredContent") or decode(nested.get("content"))
        if not any(row_sets(payload)):
            errors.append("MCP call did not return any fact rows")
    return {"passed": not errors, "reportId": report_id, "validatedRevision": revision, "mcpInvocations": len(invocations), "toolErrors": len(failures), "errors": errors}


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("transcript")
    parser.add_argument("--existing-report", default="r611102b63414acf9c018")
    options = parser.parse_args()
    report = check(options.transcript, options.existing_report)
    print(json.dumps(report, indent=2))
    sys.exit(0 if report["passed"] else 1)
