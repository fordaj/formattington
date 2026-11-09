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


class TestVersionOrdering:
    def test_version_key_is_numeric(self):
        assert _tools._version_key("9.0.0") < _tools._version_key("10.0.0")
        assert _tools._version_key("22.1.8") < _tools._version_key("23.1.3")
        assert _tools._version_key("3.14.1") == (3, 14, 1)

    def test_prefers_numeric_newest_over_lexical(self, monkeypatch, tmp_path):
        monkeypatch.setattr(_tools, "TOOLS_DIR", tmp_path)
        _make_bundled_layout(
            tmp_path, "clang-format", "9.0.0", "macos-x86_64", "clang-format"
        )
        newest = _make_bundled_layout(
            tmp_path, "clang-format", "10.0.0", "macos-x86_64", "clang-format"
        )
        with (
            patch("formattington._tools.platform.system", return_value="Darwin"),
            patch("formattington._tools.platform.machine", return_value="x86_64"),
            patch("formattington._tools.shutil.which", return_value=None),
        ):
            assert _tools.find_tool("clang-format") == str(newest)


class TestLegacyPlatformDir:
    def test_prefers_hyphenated_over_legacy_underscored(
        self, monkeypatch, tmp_path
    ):
        monkeypatch.setattr(_tools, "TOOLS_DIR", tmp_path)
        expected = _make_bundled_layout(
            tmp_path, "shfmt", "3.14.1", "macos-x86_64", "shfmt"
        )
        _make_bundled_layout(
            tmp_path, "shfmt", "3.14.1", "macos_x86_64", "shfmt"
        )
        with (
            patch("formattington._tools.platform.system", return_value="Darwin"),
            patch("formattington._tools.platform.machine", return_value="x86_64"),
            patch("formattington._tools.shutil.which", return_value=None),
        ):
            assert _tools.find_tool("shfmt") == str(expected)

    def test_falls_back_to_legacy_underscored(self, monkeypatch, tmp_path):
        monkeypatch.setattr(_tools, "TOOLS_DIR", tmp_path)
        legacy = _make_bundled_layout(
            tmp_path, "shfmt", "3.14.1", "macos_x86_64", "shfmt"
        )
        with (
            patch("formattington._tools.platform.system", return_value="Darwin"),
            patch("formattington._tools.platform.machine", return_value="x86_64"),
            patch("formattington._tools.shutil.which", return_value=None),
        ):
            assert _tools.find_tool("shfmt") == str(legacy)


class TestWindowsPathFallback:
    def test_tries_exe_when_name_not_found(self, monkeypatch, tmp_path):
        monkeypatch.setattr(_tools, "TOOLS_DIR", tmp_path)

        def _which(command):
            if command == "shfmt":
                return None
            if command == "shfmt.exe":
                return r"C:\tools\shfmt.exe"
            return None

        with (
            patch("formattington._tools.platform.system", return_value="Windows"),
            patch("formattington._tools.platform.machine", return_value="AMD64"),
            patch(
                "formattington._tools.shutil.which", side_effect=_which
            ) as mock_which,
        ):
            assert _tools.find_tool("shfmt") == r"C:\tools\shfmt.exe"
            calls = [call.args[0] for call in mock_which.call_args_list]
            assert calls == ["shfmt", "shfmt.exe"]


class TestVerboseResolution:
    def test_verbose_reports_bundled(self, monkeypatch, tmp_path, capsys):
        monkeypatch.setattr(_tools, "TOOLS_DIR", tmp_path)
        expected = _make_bundled_layout(
            tmp_path, "shfmt", "3.14.1", "macos-x86_64", "shfmt"
        )
        with (
            patch("formattington._tools.platform.system", return_value="Darwin"),
            patch("formattington._tools.platform.machine", return_value="x86_64"),
            patch("formattington._tools.shutil.which", return_value=None),
        ):
            assert _tools.find_tool("shfmt", verbose=True) == str(expected)
        err = capsys.readouterr().err
        assert "resolved to" in err
        assert str(expected) in err
        assert "(bundled)" in err

    def test_verbose_reports_path(self, monkeypatch, tmp_path, capsys):
        monkeypatch.setattr(_tools, "TOOLS_DIR", tmp_path)
        with (
            patch("formattington._tools.platform.system", return_value="Darwin"),
            patch("formattington._tools.platform.machine", return_value="x86_64"),
            patch(
                "formattington._tools.shutil.which",
                return_value="/usr/local/bin/shfmt",
            ),
        ):
            assert (
                _tools.find_tool("shfmt", verbose=True)
                == "/usr/local/bin/shfmt"
            )
        err = capsys.readouterr().err
        assert "resolved to" in err
        assert "(PATH)" in err

    def test_verbose_via_environment(self, monkeypatch, tmp_path, capsys):
        monkeypatch.setattr(_tools, "TOOLS_DIR", tmp_path)
        monkeypatch.setenv("FORMATTINGTON_VERBOSE", "1")
        expected = _make_bundled_layout(
            tmp_path, "shfmt", "3.14.1", "macos-x86_64", "shfmt"
        )
        with (
            patch("formattington._tools.platform.system", return_value="Darwin"),
            patch("formattington._tools.platform.machine", return_value="x86_64"),
            patch("formattington._tools.shutil.which", return_value=None),
        ):
            assert _tools.find_tool("shfmt") == str(expected)
        assert "resolved to" in capsys.readouterr().err

    def test_not_verbose_by_default(self, monkeypatch, tmp_path, capsys):
        monkeypatch.setattr(_tools, "TOOLS_DIR", tmp_path)
        monkeypatch.delenv("FORMATTINGTON_VERBOSE", raising=False)
        _make_bundled_layout(tmp_path, "shfmt", "3.14.1", "macos-x86_64", "shfmt")
        with (
            patch("formattington._tools.platform.system", return_value="Darwin"),
            patch("formattington._tools.platform.machine", return_value="x86_64"),
            patch("formattington._tools.shutil.which", return_value=None),
        ):
            _tools.find_tool("shfmt")
        assert capsys.readouterr().err == ""


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

    def test_capture_output_reports_changes(self, capsys):
        with patch(
            "formattington._tools.subprocess.run",
            return_value=self._fake_result(stdout="changed.sh\n", returncode=0),
        ):
            assert _tools.run_tool("/fake/shfmt", ["-l"], capture_output=True) == 1
            captured = capsys.readouterr()
            assert "changed.sh" in captured.out
            assert "formatting changes required" in captured.err

    def test_capture_output_clean(self):
        with patch(
            "formattington._tools.subprocess.run",
            return_value=self._fake_result(stdout="", returncode=0),
        ):
            assert _tools.run_tool("/fake/shfmt", ["-l"], capture_output=True) == 0

    def test_capture_output_propagates_nonzero(self):
        with patch(
            "formattington._tools.subprocess.run",
            return_value=self._fake_result(stdout="", stderr="boom", returncode=2),
        ):
            assert _tools.run_tool("/fake/shfmt", ["-l"], capture_output=True) == 2

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


class TestSharedPredicates:
    @pytest.mark.parametrize(
        "args",
        [
            ["--help"],
            ["-help"],
            ["--version"],
            ["-version"],
            ["-h"],
            ["-V"],
            ["file.cpp", "--help"],
        ],
    )
    def test_is_informational_true(self, args):
        assert _tools.is_informational(list(args)) is True

    @pytest.mark.parametrize(
        "args",
        [
            [],
            ["file.cpp"],
            ["--check"],
            ["--style=file"],
        ],
    )
    def test_is_informational_false(self, args):
        assert _tools.is_informational(list(args)) is False

    @pytest.mark.parametrize(
        ("check", "dry_run"),
        [(False, False), (True, False), (False, True)],
    )
    def test_ensure_valid_mode_ok(self, check, dry_run):
        _tools.ensure_valid_mode(check=check, dry_run=dry_run)

    def test_ensure_valid_mode_rejects_both(self):
        with pytest.raises(ValueError, match="cannot both be enabled"):
            _tools.ensure_valid_mode(check=True, dry_run=True)
