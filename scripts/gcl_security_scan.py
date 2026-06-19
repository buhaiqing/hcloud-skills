#!/usr/bin/env python3
"""Shared GCL secret-leak scanner for trace and summary artifacts."""

from __future__ import annotations

import re
from typing import Any

from gcl_runner import SECRET_PATTERNS

SCANNED_TEXT_FIELDS = {
    "request",
    "command",
    "result_excerpt",
    "operation",
    "user_request",
    "summary",
    "final_state",
    "raw_response",
}

EXTRA_PATTERNS: tuple[tuple[str, re.Pattern[str]], ...] = (
    ("bearer_token", re.compile(r"Bearer\s+[A-Za-z0-9._\-]{20,}", re.I)),
    ("authorization_header", re.compile(r"Authorization\s*[:=]\s*['\"]?[^\s'\"]+", re.I)),
    ("private_key_block", re.compile(r"-----BEGIN (?:RSA |EC |DSA |OPENSSH |PGP )?PRIVATE KEY-----")),
    ("password_assignment", re.compile(r"(?i)password\s*[:=]\s*['\"]?[^'\"\s]{6,}")),
    ("api_key_assignment", re.compile(r"(?i)(?:api[_-]?key|secret[_-]?key)\s*[:=]\s*['\"]?[A-Za-z0-9._\-/+=]{16,}")),
)


def is_scanned_text(value: str, field: str) -> bool:
    return field in SCANNED_TEXT_FIELDS or bool(value) and len(value) <= 200_000


def strings_in(value: Any, prefix: str = "") -> list[tuple[str, str]]:
    out: list[tuple[str, str]] = []
    if isinstance(value, dict):
        for key, item in value.items():
            child = f"{prefix}.{key}" if prefix else str(key)
            if isinstance(item, str):
                out.append((child, item))
            else:
                out.extend(strings_in(item, child))
    elif isinstance(value, list):
        for index, item in enumerate(value):
            child = f"{prefix}[{index}]"
            if isinstance(item, str):
                out.append((child, item))
            else:
                out.extend(strings_in(item, child))
    return out


def scan_text(text: str) -> list[str]:
    findings: list[str] = []
    if "<masked>" in text:
        return findings
    for pattern in SECRET_PATTERNS:
        if pattern.search(text):
            findings.append(pattern.pattern)
    for label, pattern in EXTRA_PATTERNS:
        if pattern.search(text):
            findings.append(f"extra:{label}")
    return findings


def scan_payload(payload: Any) -> list[dict[str, str]]:
    findings: list[dict[str, str]] = []
    for field, value in strings_in(payload):
        if not is_scanned_text(value, field):
            continue
        for match in scan_text(value):
            findings.append({"field": field, "pattern": match})
    return findings
