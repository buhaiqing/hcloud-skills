#!/usr/bin/env python3
"""Unit tests for scripts/check_py310_compat.py."""

from __future__ import annotations

import contextlib
import io
import json
import shutil
import sys
import tempfile
import textwrap
import unittest
from pathlib import Path

_HERE = Path(__file__).resolve().parent
if str(_HERE) not in sys.path:
    sys.path.insert(0, str(_HERE))

import check_py310_compat as cpc  # noqa: E402

ROOT = Path(__file__).resolve().parents[1]


def write_script(root: Path, name: str, body: str) -> Path:
    scripts_dir = root / "scripts"
    scripts_dir.mkdir(parents=True, exist_ok=True)
    path = scripts_dir / name
    path.write_text(textwrap.dedent(body), encoding="utf-8")
    return path


class ResolvePythonBinTests(unittest.TestCase):
    def test_default_finds_python310_or_python3_10(self) -> None:
        resolved = cpc.resolve_python_bin(None)
        self.assertTrue(shutil.which(resolved), resolved)

    def test_explicit_missing_raises(self) -> None:
        with self.assertRaises(SystemExit):
            cpc.resolve_python_bin("definitely-not-a-python-binary-xyz")


class DiscoverTests(unittest.TestCase):
    def test_discover_with_explicit_missing_filtered(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "scripts").mkdir()
            (root / "scripts" / "ok.py").write_text("pass\n", encoding="utf-8")
            missing = root / "scripts" / "absent.py"
            scripts = cpc.discover_scripts(root, [missing, root / "scripts" / "ok.py"])
            self.assertEqual([path.name for path in scripts], ["ok.py"])

    def test_discover_defaults_to_scripts_dir(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write_script(root, "alpha.py", "x = 1\n")
            write_script(root, "beta.py", "x = 2\n")
            self.assertEqual(
                [path.name for path in cpc.discover_scripts(root, [])],
                ["alpha.py", "beta.py"],
            )


class CompileTests(unittest.TestCase):
    def test_clean_script_passes(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            script = write_script(root, "clean.py", "x: int = 1\n")
            ok, message = cpc.compile_one(cpc.resolve_python_bin(None), script)
            self.assertTrue(ok, message)

    def test_pep_695_syntax_fails_on_3_10(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            script = write_script(
                root,
                "pep695.py",
                """\
                type Alias = int
                def f(x: Alias) -> int:
                    return x
                """,
            )
            ok, message = cpc.compile_one(cpc.resolve_python_bin(None), script)
            self.assertFalse(ok)
            self.assertTrue(message, "expected a non-empty error from python3.10")


class ImportTests(unittest.TestCase):
    """Lock the contract that the import dry-run catches 3.11-only names
    (``from datetime import UTC`` is the bug the gate was added for). A
    failure here means a 3.11+ symbol silently leaked into ``scripts/``."""

    def test_clean_script_imports_on_3_10(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            script = write_script(
                root,
                "clean.py",
                "import json\nimport pathlib\nx = json.dumps({'k': 1})\n",
            )
            ok, message = cpc.import_one(cpc.resolve_python_bin(None), script)
            self.assertTrue(ok, message)

    def test_datetime_utc_import_fails_on_3_10(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            script = write_script(
                root,
                "uses_utc.py",
                "from datetime import UTC, datetime\nx = datetime.now(UTC)\n",
            )
            ok, message = cpc.import_one(cpc.resolve_python_bin(None), script)
            self.assertFalse(ok)
            self.assertIn("UTC", message)

    def test_tomllib_import_fails_on_3_10(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            script = write_script(root, "uses_tomllib.py", "import tomllib\n")
            ok, message = cpc.import_one(cpc.resolve_python_bin(None), script)
            self.assertFalse(ok)
            self.assertIn("tomllib", message)

    def test_no_import_check_flag_skips_dry_run(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write_script(
                root,
                "uses_utc.py",
                "from datetime import UTC, datetime\nx = datetime.now(UTC)\n",
            )
            old_argv = sys.argv
            try:
                sys.argv = [
                    "check_py310_compat.py",
                    "--root",
                    str(root),
                    "--no-import-check",
                    "--json",
                ]
                with contextlib.redirect_stdout(io.StringIO()) as stdout:
                    rc = cpc.main()
                payload = json.loads(stdout.getvalue())
                # Syntax check passes; import check is opt-out.
                self.assertEqual(rc, 0)
                self.assertTrue(payload["results"][0]["ok"])
            finally:
                sys.argv = old_argv


class CliTests(unittest.TestCase):
    def test_main_clean_repo_passes(self) -> None:
        old_argv = sys.argv
        try:
            sys.argv = ["check_py310_compat.py", "--root", str(ROOT)]
            with contextlib.redirect_stdout(io.StringIO()):
                rc = cpc.main()
            self.assertEqual(rc, 0)
        finally:
            sys.argv = old_argv

    def test_main_detects_incompatible_script(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write_script(
                root,
                "broken.py",
                """\
                type Alias = int
                def f(x: Alias) -> int:
                    return x
                """,
            )
            old_argv = sys.argv
            try:
                sys.argv = ["check_py310_compat.py", "--root", str(root), "--json"]
                with contextlib.redirect_stdout(io.StringIO()) as stdout:
                    rc = cpc.main()
                payload = json.loads(stdout.getvalue())
                self.assertEqual(rc, 1)
                self.assertFalse(payload["ok"])
                self.assertEqual(len(payload["results"]), 1)
                self.assertFalse(payload["results"][0]["ok"])
            finally:
                sys.argv = old_argv


if __name__ == "__main__":
    unittest.main()
