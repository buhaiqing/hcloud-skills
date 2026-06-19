#!/usr/bin/env python3
"""Unit tests for scripts/install_git_hook.py."""

from __future__ import annotations

import contextlib
import io
import sys
import tempfile
import unittest
from pathlib import Path

_HERE = Path(__file__).resolve().parent
if str(_HERE) not in sys.path:
    sys.path.insert(0, str(_HERE))

import install_git_hook as igh  # noqa: E402


def make_fake_repo(tmp: Path) -> Path:
    """Create a minimal directory with .git/ and .githooks/pre-commit."""
    repo = Path(tmp) / "repo"
    repo.mkdir()
    (repo / ".git" / "hooks").mkdir(parents=True)
    githooks = repo / ".githooks"
    githooks.mkdir()
    (githooks / "pre-commit").write_text("#!/usr/bin/env bash\necho hook\n", encoding="utf-8")
    return repo


class InstallTests(unittest.TestCase):
    def test_install_copies_and_makes_executable(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            repo = make_fake_repo(tmp)
            rc = igh.install(repo)
            self.assertEqual(rc, 0)
            dest = repo / ".git" / "hooks" / "pre-commit"
            self.assertTrue(dest.is_file())
            self.assertEqual(dest.read_text(encoding="utf-8"), "#!/usr/bin/env bash\necho hook\n")
            mode = dest.stat().st_mode & 0o777
            self.assertTrue(mode & 0o111, f"hook should be executable, got mode {oct(mode)}")

    def test_install_is_idempotent(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            repo = make_fake_repo(tmp)
            self.assertEqual(igh.install(repo), 0)
            (repo / ".git" / "hooks" / "pre-commit").write_text("stale", encoding="utf-8")
            self.assertEqual(igh.install(repo), 0)
            self.assertEqual(
                (repo / ".git" / "hooks" / "pre-commit").read_text(encoding="utf-8"),
                "#!/usr/bin/env bash\necho hook\n",
            )

    def test_install_missing_source_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            repo = make_fake_repo(tmp)
            (repo / ".githooks" / "pre-commit").unlink()
            self.assertEqual(igh.install(repo), 1)

    def test_uninstall_removes_hook(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            repo = make_fake_repo(tmp)
            igh.install(repo)
            self.assertEqual(igh.uninstall(repo), 0)
            self.assertFalse((repo / ".git" / "hooks" / "pre-commit").exists())

    def test_uninstall_when_missing_is_ok(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            repo = make_fake_repo(tmp)
            self.assertEqual(igh.uninstall(repo), 0)

    def test_check_reports_missing(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            repo = make_fake_repo(tmp)
            self.assertEqual(igh.check(repo), 1)

    def test_check_reports_installed(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            repo = make_fake_repo(tmp)
            igh.install(repo)
            self.assertEqual(igh.check(repo), 0)


class CliTests(unittest.TestCase):
    def test_main_install_argument(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            repo = make_fake_repo(tmp)
            old_argv = sys.argv
            try:
                sys.argv = ["install_git_hook.py", "--root", str(repo), "--check"]
                with contextlib.redirect_stdout(io.StringIO()) as stdout:
                    rc = igh.main()
                self.assertEqual(rc, 1)
                self.assertIn("MISSING", stdout.getvalue())
            finally:
                sys.argv = old_argv

    def test_main_uninstall_argument(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            repo = make_fake_repo(tmp)
            igh.install(repo)
            old_argv = sys.argv
            try:
                sys.argv = ["install_git_hook.py", "--root", str(repo), "--uninstall"]
                with contextlib.redirect_stdout(io.StringIO()):
                    rc = igh.main()
                self.assertEqual(rc, 0)
                self.assertFalse((repo / ".git" / "hooks" / "pre-commit").exists())
            finally:
                sys.argv = old_argv


if __name__ == "__main__":
    unittest.main()
