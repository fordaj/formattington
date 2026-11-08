"""Tests for platform detection, tool discovery, and run_tool."""

from __future__ import annotations

from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest

from formattington import _tools


class TestGetPlatformDir:
    @pytest.mark.parametrize(
        ("system", "machine", "expected"),
        [
            ("Windows", "AMD64", "windows-x86_64"),
            ("Windows", "x86_64", "windows-x86_64"),
            ("Darwin", "arm64", "macos-arm64"),
            ("Darwin", "x86_64", "macos-x86_64"),
            ("Linux", "x86_64", "linux-x86_64"),
            ("Linux", "aarch64", "linux-arm64"),
        ],
    )
    def test_known_platforms(self, system, machine, expected):
        with (
            patch("formattington._tools.platform.system", return_value=system),
            patch("formattington._tools.platform.machine", return_value=machine),
        ):
            assert _tools.get_platform_dir() == expected

    def test_unsupported_platform_raises(self):
        with (
            patch("formattington._tools.platform.system", return_value="FreeBSD"),
            patch("formattington._tools.platform.machine", return_value="sparc"),
        ):
            with pytest.raises(RuntimeError, match="Unsupported platform"):
                _tools.get_platform_dir()

    def test_unsupported_architecture_raises(self):
        with (
            patch("formattington._tools.platform.system", return_value="Linux"),
            patch("formattington._tools.platform.machine", return_value="riscv"),
        ):
            with pytest.raises(RuntimeError, match="Unsupported platform"):
                _tools.get_platform_dir()


def _make_bundled_layout(tools_dir: Path, name: str, version: str, platform_dir: str, exe: str) -> Path:
    bundled = tools_dir / name / f"v{version}" / platform_dir / exe
    bundled.parent.mkdir(parents=True, exist_ok=True)
    bundled.touch()
    return bundled


class TestFindTool:
    def test_prefers_bundled_tool(self, monkeypatch, tmp_path):
        monkeypatch.setattr(_tools, "TOOLS_DIR", tmp_path)
        expected = _make_bundled_layout(tmp_path, "shfmt", "3.14.1", "macos-x86_64", "shfmt")
        with (
            patch("formattington._tools.platform.system", return_value="Darwin"),
            patch("formattington._tools.platform.machine", return_value="x86_64"),
            patch("formattington._tools.shutil.which", return_value="/usr/local/bin/shfmt"),
        ):
            assert _tools.find_tool("shfmt") == str(expected)

    def test_respects_explicit_version_with_v_prefix(self, monkeypatch, tmp_path):
        monkeypatch.setattr(_tools, "TOOLS_DIR", tmp_path)
        expected = _make_bundled_layout(tmp_path, "shfmt", "3.14.1", "macos-x86_64", "shfmt")
        with (
            patch("formattington._tools.platform.system", return_value="Darwin"),
            patch("formattington._tools.platform.machine", return_value="x86_64"),
        ):
            assert _tools.find_tool("shfmt", version="v3.14.1") == str(expected)

    def test_respects_explicit_version_without_prefix(self, monkeypatch, tmp_path):
        monkeypatch.setattr(_tools, "TOOLS_DIR", tmp_path)
        expected = _make_bundled_layout(tmp_path, "shfmt", "3.14.1", "macos-x86_64", "shfmt")
        with (
            patch("formattington._tools.platform.system", return_value="Darwin"),
            patch("formattington._tools.platform.machine", return_value="x86_64"),
        ):
            assert _tools.find_tool("shfmt", version="3.14.1") == str(expected)

    def test_prefers_newest_version_dir(self, monkeypatch, tmp_path):
        monkeypatch.setattr(_tools, "TOOLS_DIR", tmp_path)
        _make_bundled_layout(tmp_path, "clang-format", "22.1.8", "macos-x86_64", "clang-format")
        newest = _make_bundled_layout(tmp_path, "clang-format", "23.1.3", "macos-x86_64", "clang-format")
        with (
            patch("formattington._tools.platform.system", return_value="Darwin"),
            patch("formattington._tools.platform.machine", return_value="x86_64"),
        ):
            assert _tools.find_tool("clang-format") == str(newest)

    def test_falls_back_to_path(self, monkeypatch, tmp_path):
        monkeypatch.setattr(_tools, "TOOLS_DIR", tmp_path)
        # No bundled tools present; PATH has the tool.
        with (
            patch("formattington._tools.platform.system", return_value="Darwin"),
            patch("formattington._tools.platform.machine", return_value="x86_64"),
            patch("formattington._tools.shutil.which", return_value="/usr/local/bin/shfmt"),
        ):
            assert _tools.find_tool("shfmt") == "/usr/local/bin/shfmt"

    def test_raises_when_missing_everywhere(self, monkeypatch, tmp_path):
        monkeypatch.setattr(_tools, "TOOLS_DIR", tmp_path)
        with (
            patch("formattington._tools.platform.system", return_value="Darwin"),
            patch("formattington._tools.platform.machine", return_value="x86_64"),
            patch("formattington._tools.shutil.which", return_value=None),
        ):
            with pytest.raises(FileNotFoundError, match="Could not find 'shfmt'"):
                _tools.find_tool("shfmt")

    def test_windows_looks_for_exe(self, monkeypatch, tmp_path):
        monkeypatch.setattr(_tools, "TOOLS_DIR", tmp_path)
        expected = _make_bundled_layout(tmp_path, "shfmt", "3.14.1", "windows-x86_64", "shfmt.exe")
        with (
            patch("formattington._tools.platform.system", return_value="Windows"),
            patch("formattington._tools.platform.machine", return_value="AMD64"),
        ):
            assert _tools.find_tool("shfmt") == str(expected)

    def test_skips_version_dirs_for_other_platforms(self, monkeypatch, tmp_path):
        monkeypatch.setattr(_tools, "TOOLS_DIR", tmp_path)
        # Only a linux bundle exists; darwin request should fall back to PATH.
        _make_bundled_layout(tmp_path, "shfmt", "3.14.1", "linux-x86_64", "shfmt")
        with (
            patch("formattington._tools.platform.system", return_value="Darwin"),
            patch("formattington._tools.platform.machine", return_value="x86_64"),
            patch("formattington._tools.shutil.which", return_value="/usr/local/bin/shfmt"),
        ):
            assert _tools.find_tool("shfmt") == "/usr/local/bin/shfmt"


class TestRunTool:
    def _fake_result(self, *, stdout: str = "", stderr: str = "", returncode: int = 0):
        result = MagicMock()
        result.stdout = stdout
        result.stderr = stderr
        result.returncode = returncode
        return result

    def test_returns_subprocess_exit_code(self):
        with patch(
            "formattington._tools.subprocess.run",
            return_value=self._fake_result(returncode=3),
        ) as mock_run:
            assert _tools.run_tool("/fake/shfmt", ["-w"]) == 3
            mock_run.assert_called_once_with(
                ["/fake/shfmt", "-w"],
                capture_output=False,
                text=True,
                check=False,
            )

    def test_check_output_reports_changes(self, capsys):
        with patch(
            "formattington._tools.subprocess.run",
            return_value=self._fake_result(stdout="changed.sh\n", returncode=0),
        ):
            assert _tools.run_tool("/fake/shfmt", ["-l"], check_output=True) == 1
            captured = capsys.readouterr()
            assert "changed.sh" in captured.out
            assert "formatting changes required" in captured.err

    def test_check_output_clean(self):
        with patch(
            "formattington._tools.subprocess.run",
            return_value=self._fake_result(stdout="", returncode=0),
        ):
            assert _tools.run_tool("/fake/shfmt", ["-l"], check_output=True) == 0

    def test_check_output_propagates_nonzero(self):
        with patch(
            "formattington._tools.subprocess.run",
            return_value=self._fake_result(stdout="", stderr="boom", returncode=2),
        ):
            assert _tools.run_tool("/fake/shfmt", ["-l"], check_output=True) == 2

    def test_passes_args_as_list_not_shell_string(self):
        with patch(
            "formattington._tools.subprocess.run",
            return_value=self._fake_result(returncode=0),
        ) as mock_run:
            args = ["file with spaces.sh", "another file.sh"]
            _tools.run_tool("/fake/shfmt", ["-w", *args])
            cmd = mock_run.call_args[0][0]
            assert cmd == ["/fake/shfmt", "-w", "file with spaces.sh", "another file.sh"]
            # Spaces are preserved as single elements; no shell joining.
            assert mock_run.call_args[1].get("shell") is None
