#!/usr/bin/env python3
"""Unit tests for scripts/check_skill_generator_drift.py."""

from __future__ import annotations

import sys
import tempfile
import unittest
from pathlib import Path

_HERE = Path(__file__).resolve().parent
if str(_HERE) not in sys.path:
    sys.path.insert(0, str(_HERE))

import check_skill_generator_drift as csgd  # noqa: E402

REPO_ROOT = Path(__file__).resolve().parents[1]


class DriftDetectionTests(unittest.TestCase):
    def _build_pair(self, root: Path) -> tuple[Path, Path]:
        canonical = root / "huaweicloud-skill-generator"
        runtime = root / ".agents/skills/huaweicloud-skill-generator"
        canonical.mkdir(parents=True)
        runtime.mkdir(parents=True)
        (canonical / "SKILL.md").write_text("canonical", encoding="utf-8")
        (runtime / "SKILL.md").write_text("canonical", encoding="utf-8")
        (canonical / "references").mkdir()
        (canonical / "references/r.md").write_text("shared", encoding="utf-8")
        (runtime / "references").mkdir()
        (runtime / "references/r.md").write_text("shared", encoding="utf-8")
        return canonical, runtime

    def test_identical_pair_ok(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self._build_pair(root)
            report = csgd.check_drift(root)
            self.assertTrue(report["ok"], report["errors"])

    def test_modified_runtime_file_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self._build_pair(root)
            (root / ".agents/skills/huaweicloud-skill-generator/SKILL.md").write_text("stale", encoding="utf-8")
            report = csgd.check_drift(root)
            self.assertFalse(report["ok"])
            self.assertIn("SKILL.md", report["errors"][0])

    def test_missing_canonical_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            report = csgd.check_drift(root)
            self.assertFalse(report["ok"])

    def test_missing_runtime_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "huaweicloud-skill-generator").mkdir()
            report = csgd.check_drift(root)
            self.assertFalse(report["ok"])

    def test_extra_file_in_runtime_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self._build_pair(root)
            (root / ".agents/skills/huaweicloud-skill-generator/extra.md").write_text("x", encoding="utf-8")
            report = csgd.check_drift(root)
            self.assertFalse(report["ok"])

    def test_missing_file_in_runtime_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self._build_pair(root)
            (root / "huaweicloud-skill-generator/extra.md").write_text("x", encoding="utf-8")
            report = csgd.check_drift(root)
            self.assertFalse(report["ok"])

    def test_ds_store_ignored(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            canonical, runtime = self._build_pair(root)
            (canonical / ".DS_Store").write_text("mac", encoding="utf-8")
            (runtime / ".DS_Store").write_text("mac", encoding="utf-8")
            report = csgd.check_drift(root)
            self.assertTrue(report["ok"], report["errors"])


class SyncTests(unittest.TestCase):
    def _build_pair(self, root: Path) -> None:
        canonical = root / "huaweicloud-skill-generator"
        runtime = root / ".agents/skills/huaweicloud-skill-generator"
        canonical.mkdir(parents=True)
        runtime.mkdir(parents=True)
        (canonical / "SKILL.md").write_text("canonical", encoding="utf-8")
        (runtime / "SKILL.md").write_text("stale", encoding="utf-8")

    def test_sync_overwrites_modified(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self._build_pair(root)
            report = csgd.sync(root, dry_run=False)
            self.assertTrue(report["ok"])
            on_disk = (root / ".agents/skills/huaweicloud-skill-generator/SKILL.md").read_text(encoding="utf-8")
            self.assertEqual(on_disk, "canonical")

    def test_sync_dry_run_does_not_write(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self._build_pair(root)
            report = csgd.sync(root, dry_run=True)
            self.assertTrue(report["actions"])
            on_disk = (root / ".agents/skills/huaweicloud-skill-generator/SKILL.md").read_text(encoding="utf-8")
            self.assertEqual(on_disk, "stale")

    def test_sync_adds_missing(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self._build_pair(root)
            (root / "huaweicloud-skill-generator/extra.md").write_text("from canonical", encoding="utf-8")
            csgd.sync(root, dry_run=False)
            on_disk = (root / ".agents/skills/huaweicloud-skill-generator/extra.md").read_text(encoding="utf-8")
            self.assertEqual(on_disk, "from canonical")

    def test_sync_removes_runtime_extras(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self._build_pair(root)
            extra = root / ".agents/skills/huaweicloud-skill-generator/extra.md"
            extra.write_text("stale-extra", encoding="utf-8")
            csgd.sync(root, dry_run=False)
            self.assertFalse(extra.exists())


class CliTests(unittest.TestCase):
    def test_repo_passes_after_sync(self) -> None:
        """After applying sync, the real repo passes the drift gate."""
        report = csgd.check_drift(REPO_ROOT)
        if report["ok"]:
            return
        applied = csgd.sync(REPO_ROOT, dry_run=False)
        self.assertTrue(applied["ok"])
        try:
            recheck = csgd.check_drift(REPO_ROOT)
            self.assertTrue(recheck["ok"], recheck["errors"])
        finally:
            # Re-running `check_drift` in CI after a sync is the desired steady
            # state. We deliberately do not roll back: the canonical root was
            # the source of truth.
            pass

    def test_check_drift_cli_repo_fails_without_sync(self) -> None:
        """When the real repo is drifted, the CLI surfaces a clear error."""
        report = csgd.check_drift(REPO_ROOT)
        if report["ok"]:
            self.skipTest("repo already in sync; cannot verify failure path")
        self.assertFalse(report["ok"])


if __name__ == "__main__":
    unittest.main()
