#!/usr/bin/env python3
"""Unit tests for scripts/check_references_link_health.py."""

from __future__ import annotations

import contextlib
import io
import json
import sys
import tempfile
import textwrap
import unittest
from pathlib import Path

_HERE = Path(__file__).resolve().parent
if str(_HERE) not in sys.path:
    sys.path.insert(0, str(_HERE))

import check_references_link_health as crlh  # noqa: E402


def write(path: Path, body: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(textwrap.dedent(body).lstrip("\n"), encoding="utf-8")


class SlugifierTests(unittest.TestCase):
    def test_basic_lowercase_and_dash(self) -> None:
        self.assertEqual(crlh.slugify_anchor("Hello World"), "hello-world")

    def test_punctuation_dropped(self) -> None:
        self.assertEqual(crlh.slugify_anchor("3. FinOps (财务运营)"), "3-finops-财务运营")

    def test_digits_kept(self) -> None:
        self.assertEqual(crlh.slugify_anchor("21 安全支柱"), "21-安全支柱")

    def test_inline_code_stripped(self) -> None:
        self.assertEqual(crlh.slugify_anchor("Use `hcloud`"), "use-hcloud")


class HeadingInventoryTests(unittest.TestCase):
    def test_inventory_collects_headings(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "doc.md"
            write(
                path,
                """
                # Title
                ## 3. FinOps (财务运营)
                ### 3.1 Cost Visibility
            """,
            )
            inventory = crlh.inventory_headings(path)
            self.assertIn("title", inventory.anchors)
            self.assertIn("3-finops-财务运营", inventory.anchors)
            self.assertIn("31-cost-visibility", inventory.anchors)


class CheckFileTests(unittest.TestCase):
    def test_clean_references_dir_passes(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write(
                root / "huaweicloud-foo-ops/references/page-a.md",
                """
                # A
                See [B](page-b.md#section) for details.
            """,
            )
            write(
                root / "huaweicloud-foo-ops/references/page-b.md",
                """
                # B
                ## Section
                Body.
            """,
            )
            findings = crlh.check_file(root, root / "huaweicloud-foo-ops/references/page-a.md", {})
            self.assertEqual(findings, [], findings)

    def test_missing_anchor_is_error(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write(
                root / "huaweicloud-foo-ops/references/page-a.md",
                """
                # A
                See [B](page-b.md#missing) for details.
            """,
            )
            write(
                root / "huaweicloud-foo-ops/references/page-b.md",
                """
                # B
                ## Section
            """,
            )
            findings = crlh.check_file(root, root / "huaweicloud-foo-ops/references/page-a.md", {})
            self.assertEqual(len(findings), 1)
            self.assertEqual(findings[0].severity, "error")
            self.assertIn("missing anchor", findings[0].reason)

    def test_missing_file_is_error(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write(
                root / "huaweicloud-foo-ops/references/page-a.md",
                """
                # A
                [ghost](missing.md)
            """,
            )
            findings = crlh.check_file(root, root / "huaweicloud-foo-ops/references/page-a.md", {})
            self.assertEqual(len(findings), 1)
            self.assertEqual(findings[0].severity, "error")

    def test_sibling_bare_name_emits_warning(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write(
                root / "huaweicloud-foo-ops/references/page-a.md",
                """
                # A
                See [B](page-b) for details.
            """,
            )
            write(
                root / "huaweicloud-foo-ops/references/page-b.md",
                """
                # B
            """,
            )
            findings = crlh.check_file(root, root / "huaweicloud-foo-ops/references/page-a.md", {})
            self.assertEqual(len(findings), 1)
            self.assertEqual(findings[0].severity, "warning")
            self.assertIn("page-b.md", findings[0].reason)

    def test_in_page_anchor_resolves(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write(
                root / "huaweicloud-foo-ops/references/page-a.md",
                """
                # A
                ## Section
                Jump to [Section](#section).
            """,
            )
            findings = crlh.check_file(root, root / "huaweicloud-foo-ops/references/page-a.md", {})
            self.assertEqual(findings, [], findings)

    def test_cross_skill_missing_is_error(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write(
                root / "huaweicloud-foo-ops/references/page-a.md",
                """
                # A
                See [iam skill](huaweicloud-iam-ops/SKILL.md).
            """,
            )
            findings = crlh.check_file(root, root / "huaweicloud-foo-ops/references/page-a.md", {})
            self.assertEqual(len(findings), 1)
            self.assertEqual(findings[0].severity, "error")
            self.assertIn("cross-skill", findings[0].reason)

    def test_http_links_are_ignored(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write(
                root / "huaweicloud-foo-ops/references/page-a.md",
                """
                # A
                See [docs](https://example.com) and [GH](http://foo).
            """,
            )
            findings = crlh.check_file(root, root / "huaweicloud-foo-ops/references/page-a.md", {})
            self.assertEqual(findings, [], findings)


class CollectTests(unittest.TestCase):
    def test_full_repo_clean_baseline(self) -> None:
        root = Path(__file__).resolve().parents[1]
        report = crlh.collect(root)
        self.assertTrue(report["ok"], report["findings"])
        self.assertEqual(report["summary"]["errors"], 0)
        self.assertGreater(report["summary"]["files_scanned"], 100)

    def test_repo_files_scanned_are_in_references(self) -> None:
        root = Path(__file__).resolve().parents[1]
        report = crlh.collect(root)
        for rel in report["files"]:
            assert isinstance(rel, str)
            self.assertIn("/references/", rel, rel)


class CliTests(unittest.TestCase):
    def test_main_clean_repo_returns_0(self) -> None:
        root = Path(__file__).resolve().parents[1]
        old_argv = sys.argv
        try:
            sys.argv = ["check_references_link_health.py", "--root", str(root)]
            with contextlib.redirect_stdout(io.StringIO()) as stdout:
                rc = crlh.main()
            self.assertEqual(rc, 0)
            self.assertIn("scanned", stdout.getvalue())
        finally:
            sys.argv = old_argv

    def test_main_json_mode(self) -> None:
        root = Path(__file__).resolve().parents[1]
        old_argv = sys.argv
        try:
            sys.argv = ["check_references_link_health.py", "--root", str(root), "--json"]
            with contextlib.redirect_stdout(io.StringIO()) as stdout:
                rc = crlh.main()
            payload = json.loads(stdout.getvalue())
            self.assertIn("summary", payload)
            self.assertIn("files", payload)
            self.assertEqual(rc, 0 if payload["ok"] else 1)
        finally:
            sys.argv = old_argv


if __name__ == "__main__":
    unittest.main()
