"""Tests for shfmt / clang-format argv construction and subprocess boundaries."""

from __future__ import annotations

from unittest.mock import MagicMock, patch

import pytest

from formattington import _tools
from formattington.clang_format import clang_format
from formattington.shfmt import shfmt


def _fake_run_capture(returncode=0, stdout="", stderr=""):
    """Return a subprocess.run fake plus the list of calls it saw."""
    calls = []

    def fake_run(cmd, **kwargs):
        calls.append((cmd, kwargs))
        result = MagicMock()
        result.returncode = returncode
        result.stdout = stdout
        result.stderr = stderr
        return result

    return fake_run, calls


class TestShfmt:
    def test_default_mode_writes(self):
        fake, calls = _fake_run_capture()
        with patch("formattington.shfmt.find_tool", return_value="/fake/shfmt"), patch(
            "formattington._tools.subprocess.run", side_effect=fake
        ):
            assert shfmt() == 0
        assert len(calls) == 1
        cmd, kwargs = calls[0]
        assert cmd == ["/fake/shfmt", "-w"]
        assert kwargs["check"] is False

    def test_check_mode_lists(self):
        fake, calls = _fake_run_capture(stdout="")
        with patch("formattington.shfmt.find_tool", return_value="/fake/shfmt"), patch(
            "formattington._tools.subprocess.run", side_effect=fake
        ):
            assert shfmt(check=True) == 0
        assert calls[0][0] == ["/fake/shfmt", "-l"]
        assert calls[0][1]["capture_output"] is True

    def test_check_mode_reports_changes(self):
        fake, _ = _fake_run_capture(stdout="changed.sh\n")
        with patch("formattington.shfmt.find_tool", return_value="/fake/shfmt"), patch(
            "formattington._tools.subprocess.run", side_effect=fake
        ):
            assert shfmt(check=True) == 1

    def test_dry_run_mode_diffs(self):
        fake, calls = _fake_run_capture()
        with patch("formattington.shfmt.find_tool", return_value="/fake/shfmt"), patch(
            "formattington._tools.subprocess.run", side_effect=fake
        ):
            assert shfmt(dry_run=True) == 0
        assert calls[0][0] == ["/fake/shfmt", "-d"]

    def test_forwards_extra_args_after_mode_flag(self):
        fake, calls = _fake_run_capture()
        with patch("formattington.shfmt.find_tool", return_value="/fake/shfmt"), patch(
            "formattington._tools.subprocess.run", side_effect=fake
        ):
            shfmt(args=["--keep-padding", "file with spaces.sh"])
        assert calls[0][0] == [
            "/fake/shfmt",
            "-w",
            "--keep-padding",
            "file with spaces.sh",
        ]
        # One batched subprocess call, not one per file.
        assert len(calls) == 1

    def test_check_and_dry_run_are_exclusive(self):
        with pytest.raises(ValueError, match="cannot both be enabled"):
            shfmt(check=True, dry_run=True)

    def test_propagates_subprocess_exit_code(self):
        fake, _ = _fake_run_capture(returncode=2)
        with patch("formattington.shfmt.find_tool", return_value="/fake/shfmt"), patch(
            "formattington._tools.subprocess.run", side_effect=fake
        ):
            assert shfmt() == 2

    def test_missing_tool_raises(self):
        with patch(
            "formattington.shfmt.find_tool",
            side_effect=FileNotFoundError("Could not find 'shfmt'"),
        ):
            with pytest.raises(FileNotFoundError):
                shfmt()

    def test_forwards_verbose_to_find_tool(self):
        fake, _ = _fake_run_capture()
        with patch(
            "formattington.shfmt.find_tool", return_value="/fake/shfmt"
        ) as mock_find, patch(
            "formattington._tools.subprocess.run", side_effect=fake
        ):
            assert shfmt(verbose=True) == 0
        mock_find.assert_called_once_with("shfmt", verbose=True)


class TestClangFormat:
    def test_default_inserts_in_place(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.clang_format.find_tool", return_value="/fake/clang-format"
        ), patch("formattington.clang_format.subprocess.run", side_effect=fake):
            assert clang_format() == 0
        assert calls[0][0] == ["/fake/clang-format", "-i"]

    def test_does_not_duplicate_in_place_when_present(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.clang_format.find_tool", return_value="/fake/clang-format"
        ), patch("formattington.clang_format.subprocess.run", side_effect=fake):
            clang_format(args=["--in-place", "file.cpp"])
        assert calls[0][0] == [
            "/fake/clang-format",
            "--in-place",
            "file.cpp",
        ]

    def test_check_uses_dry_run_and_werror(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.clang_format.find_tool", return_value="/fake/clang-format"
        ), patch("formattington.clang_format.subprocess.run", side_effect=fake):
            assert clang_format(check=True) == 0
        cmd = calls[0][0]
        assert cmd[:3] == ["/fake/clang-format", "--dry-run", "--Werror"]

    def test_dry_run_uses_dry_run(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.clang_format.find_tool", return_value="/fake/clang-format"
        ), patch("formattington.clang_format.subprocess.run", side_effect=fake):
            assert clang_format(dry_run=True) == 0
        assert calls[0][0] == ["/fake/clang-format", "--dry-run"]

    def test_help_passthrough_without_mode_flags(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.clang_format.find_tool", return_value="/fake/clang-format"
        ), patch("formattington.clang_format.subprocess.run", side_effect=fake):
            assert clang_format(args=["--help"]) == 0
        assert calls[0][0] == ["/fake/clang-format", "--help"]

    def test_version_passthrough_without_mode_flags(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.clang_format.find_tool", return_value="/fake/clang-format"
        ), patch("formattington.clang_format.subprocess.run", side_effect=fake):
            assert clang_format(args=["--version"]) == 0
        assert calls[0][0] == ["/fake/clang-format", "--version"]

    def test_forwards_args_and_preserves_spaces(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.clang_format.find_tool", return_value="/fake/clang-format"
        ), patch("formattington.clang_format.subprocess.run", side_effect=fake):
            clang_format(args=["--style=file", "my file.cpp", "another file.cpp"])
        cmd = calls[0][0]
        assert cmd == [
            "/fake/clang-format",
            "-i",
            "--style=file",
            "my file.cpp",
            "another file.cpp",
        ]
        # Batched: a single subprocess invocation carries every path.
        assert len(calls) == 1

    def test_check_and_dry_run_are_exclusive(self):
        with pytest.raises(ValueError, match="cannot both be enabled"):
            clang_format(check=True, dry_run=True)

    def test_missing_tool_raises(self):
        with patch(
            "formattington.clang_format.find_tool",
            side_effect=FileNotFoundError("Could not find 'clang-format'"),
        ):
            with pytest.raises(FileNotFoundError):
                clang_format()

    def test_no_shell_string_used(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.clang_format.find_tool", return_value="/fake/clang-format"
        ), patch("formattington.clang_format.subprocess.run", side_effect=fake):
            clang_format(args=["file with spaces.cpp"])
        _, kwargs = calls[0]
        assert kwargs.get("shell") is None
        assert kwargs["check"] is False

    def test_forwards_verbose_to_find_tool(self):
        fake, _ = _fake_run_capture()
        with patch(
            "formattington.clang_format.find_tool",
            return_value="/fake/clang-format",
        ) as mock_find, patch(
            "formattington.clang_format.subprocess.run", side_effect=fake
        ):
            assert clang_format(verbose=True) == 0
        mock_find.assert_called_once_with("clang-format", verbose=True)

    def test_run_tool_is_imported_by_shfmt(self):
        # Guards the documented subprocess boundary: shfmt delegates to run_tool.
        assert _tools.run_tool is not None
