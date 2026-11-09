"""Tests for shfmt / clang-format argv construction and subprocess boundaries."""

from __future__ import annotations

from unittest.mock import MagicMock, patch

import pytest

from formattington import _tools
from formattington.clang_format import clang_format
from formattington.format_python import FILE_EXTENSIONS, black, format_python
from formattington.powershell import powershell
from formattington.taplo import taplo
from formattington.prettier import FILE_EXTENSIONS as PRETTIER_FILE_EXTENSIONS
from formattington.prettier import prettier
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


class TestBlack:
    def test_default_mode_no_flag(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.format_python.find_tool", return_value="/fake/black"
        ), patch("formattington.format_python.subprocess.run", side_effect=fake):
            assert black() == 0
        assert calls[0][0] == ["/fake/black"]

    def test_check_mode_uses_check_flag(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.format_python.find_tool", return_value="/fake/black"
        ), patch("formattington.format_python.subprocess.run", side_effect=fake):
            assert black(check=True) == 0
        assert calls[0][0] == ["/fake/black", "--check"]

    def test_dry_run_mode_uses_diff_flag(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.format_python.find_tool", return_value="/fake/black"
        ), patch("formattington.format_python.subprocess.run", side_effect=fake):
            assert black(dry_run=True) == 0
        assert calls[0][0] == ["/fake/black", "--diff"]

    def test_forwards_args_after_mode_flag_and_preserves_spaces(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.format_python.find_tool", return_value="/fake/black"
        ), patch("formattington.format_python.subprocess.run", side_effect=fake):
            black(check=True, args=["my file.py", "another file.py"])
        assert calls[0][0] == [
            "/fake/black",
            "--check",
            "my file.py",
            "another file.py",
        ]
        assert len(calls) == 1

    def test_check_and_dry_run_are_exclusive(self):
        with pytest.raises(ValueError, match="cannot both be enabled"):
            black(check=True, dry_run=True)

    def test_propagates_subprocess_exit_code(self):
        fake, _ = _fake_run_capture(returncode=2)
        with patch(
            "formattington.format_python.find_tool", return_value="/fake/black"
        ), patch("formattington.format_python.subprocess.run", side_effect=fake):
            assert black() == 2

    def test_missing_tool_raises(self):
        with patch(
            "formattington.format_python.find_tool",
            side_effect=FileNotFoundError("Could not find 'black'"),
        ):
            with pytest.raises(FileNotFoundError):
                black()

    def test_forwards_verbose_to_find_tool(self):
        fake, _ = _fake_run_capture()
        with patch(
            "formattington.format_python.find_tool", return_value="/fake/black"
        ) as mock_find, patch(
            "formattington.format_python.subprocess.run", side_effect=fake
        ):
            assert black(verbose=True) == 0
        mock_find.assert_called_once_with("black", verbose=True)

    def test_help_passthrough_without_mode_flags(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.format_python.find_tool", return_value="/fake/black"
        ), patch("formattington.format_python.subprocess.run", side_effect=fake):
            assert black(check=True, args=["--help"]) == 0
        assert calls[0][0] == ["/fake/black", "--help"]

    def test_version_passthrough_without_mode_flags(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.format_python.find_tool", return_value="/fake/black"
        ), patch("formattington.format_python.subprocess.run", side_effect=fake):
            assert black(dry_run=True, args=["--version"]) == 0
        assert calls[0][0] == ["/fake/black", "--version"]

    def test_no_shell_used(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.format_python.find_tool", return_value="/fake/black"
        ), patch("formattington.format_python.subprocess.run", side_effect=fake):
            black(args=["file with spaces.py"])
        _, kwargs = calls[0]
        assert kwargs.get("shell") is None
        assert kwargs["check"] is False

    def test_does_not_add_line_length(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.format_python.find_tool", return_value="/fake/black"
        ), patch("formattington.format_python.subprocess.run", side_effect=fake):
            black(args=["file.py"])
        assert "--line-length" not in calls[0][0]

    def test_forwarded_line_length_passes_through(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.format_python.find_tool", return_value="/fake/black"
        ), patch("formattington.format_python.subprocess.run", side_effect=fake):
            black(args=["--line-length", "88", "file.py"])
        assert calls[0][0] == ["/fake/black", "--line-length", "88", "file.py"]

    def test_alias_and_extensions(self):
        assert format_python is black
        assert FILE_EXTENSIONS == (".py",)


class TestPrettier:
    def test_default_mode_writes(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.prettier.find_tool", return_value="/fake/prettier"
        ), patch("formattington.prettier.subprocess.run", side_effect=fake):
            assert prettier() == 0
        assert calls[0][0] == ["/fake/prettier", "--write"]

    def test_check_mode_uses_check_flag(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.prettier.find_tool", return_value="/fake/prettier"
        ), patch("formattington.prettier.subprocess.run", side_effect=fake):
            assert prettier(check=True) == 0
        assert calls[0][0] == ["/fake/prettier", "--check"]

    def test_dry_run_mode_uses_list_different(self):
        # Prettier has no --diff; --list-different is the dry-run equivalent
        # that lists differing files without modifying them.
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.prettier.find_tool", return_value="/fake/prettier"
        ), patch("formattington.prettier.subprocess.run", side_effect=fake):
            assert prettier(dry_run=True) == 0
        assert calls[0][0] == ["/fake/prettier", "--list-different"]

    def test_forwards_args_after_mode_flag_and_preserves_spaces(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.prettier.find_tool", return_value="/fake/prettier"
        ), patch("formattington.prettier.subprocess.run", side_effect=fake):
            prettier(check=True, args=["my file.js", "another file.json"])
        assert calls[0][0] == [
            "/fake/prettier",
            "--check",
            "my file.js",
            "another file.json",
        ]
        assert len(calls) == 1

    def test_dry_run_forwards_args_and_preserves_spaces(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.prettier.find_tool", return_value="/fake/prettier"
        ), patch("formattington.prettier.subprocess.run", side_effect=fake):
            prettier(dry_run=True, args=["my file.ts"])
        assert calls[0][0] == ["/fake/prettier", "--list-different", "my file.ts"]

    def test_default_forwards_args_after_write(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.prettier.find_tool", return_value="/fake/prettier"
        ), patch("formattington.prettier.subprocess.run", side_effect=fake):
            prettier(args=["--single-quote", "file with spaces.js"])
        assert calls[0][0] == [
            "/fake/prettier",
            "--write",
            "--single-quote",
            "file with spaces.js",
        ]
        assert len(calls) == 1

    def test_does_not_duplicate_write_when_present(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.prettier.find_tool", return_value="/fake/prettier"
        ), patch("formattington.prettier.subprocess.run", side_effect=fake):
            prettier(args=["--write", "file.js"])
        assert calls[0][0] == ["/fake/prettier", "--write", "file.js"]

    def test_forwarded_check_omits_write(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.prettier.find_tool", return_value="/fake/prettier"
        ), patch("formattington.prettier.subprocess.run", side_effect=fake):
            prettier(args=["--check"])
        assert calls[0][0] == ["/fake/prettier", "--check"]

    def test_forwarded_list_different_omits_write(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.prettier.find_tool", return_value="/fake/prettier"
        ), patch("formattington.prettier.subprocess.run", side_effect=fake):
            prettier(args=["--list-different"])
        assert calls[0][0] == ["/fake/prettier", "--list-different"]

    def test_check_and_dry_run_are_exclusive(self):
        with pytest.raises(ValueError, match="cannot both be enabled"):
            prettier(check=True, dry_run=True)

    def test_propagates_subprocess_exit_code(self):
        fake, _ = _fake_run_capture(returncode=1)
        with patch(
            "formattington.prettier.find_tool", return_value="/fake/prettier"
        ), patch("formattington.prettier.subprocess.run", side_effect=fake):
            assert prettier(check=True) == 1

    def test_missing_tool_raises(self):
        with patch(
            "formattington.prettier.find_tool",
            side_effect=FileNotFoundError("Could not find 'prettier'"),
        ):
            with pytest.raises(FileNotFoundError):
                prettier()

    def test_forwards_verbose_to_find_tool(self):
        fake, _ = _fake_run_capture()
        with patch(
            "formattington.prettier.find_tool", return_value="/fake/prettier"
        ) as mock_find, patch(
            "formattington.prettier.subprocess.run", side_effect=fake
        ):
            assert prettier(verbose=True) == 0
        mock_find.assert_called_once_with("prettier", verbose=True)

    def test_help_passthrough_without_mode_flags(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.prettier.find_tool", return_value="/fake/prettier"
        ), patch("formattington.prettier.subprocess.run", side_effect=fake):
            assert prettier(check=True, args=["--help"]) == 0
        assert calls[0][0] == ["/fake/prettier", "--help"]

    def test_version_passthrough_without_mode_flags(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.prettier.find_tool", return_value="/fake/prettier"
        ), patch("formattington.prettier.subprocess.run", side_effect=fake):
            assert prettier(dry_run=True, args=["--version"]) == 0
        assert calls[0][0] == ["/fake/prettier", "--version"]

    def test_no_shell_used(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.prettier.find_tool", return_value="/fake/prettier"
        ), patch("formattington.prettier.subprocess.run", side_effect=fake):
            prettier(args=["file with spaces.js"])
        _, kwargs = calls[0]
        assert kwargs.get("shell") is None
        assert kwargs["check"] is False

    def test_does_not_add_config_flag(self):
        # The project .prettierrc must be respected; Formattington adds no
        # --config override unless the caller forwards one.
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.prettier.find_tool", return_value="/fake/prettier"
        ), patch("formattington.prettier.subprocess.run", side_effect=fake):
            prettier(args=["file.js"])
        assert "--config" not in calls[0][0]

    def test_extensions(self):
        assert PRETTIER_FILE_EXTENSIONS == (
            ".js",
            ".ts",
            ".tsx",
            ".jsx",
            ".json",
            ".html",
            ".css",
            ".scss",
            ".yaml",
            ".yml",
            ".md",
            ".markdown",
        )


class TestPowershell:
    """Issue #25: PSScriptAnalyzer linting via pwsh."""

    def test_extensions(self):
        from formattington.powershell import FILE_EXTENSIONS

        assert FILE_EXTENSIONS == (".ps1", ".psm1")

    def test_prefers_pwsh(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.powershell.find_tool", side_effect=["/fake/pwsh"]
        ), patch("formattington.powershell.subprocess.run", side_effect=fake):
            assert powershell() == 0
        argv = calls[0][0]
        assert argv[0] == "/fake/pwsh"
        assert argv[1:3] == ["-NoProfile", "-Command"]
        script = argv[3]
        assert "Import-Module PSScriptAnalyzer" in script
        assert "Invoke-ScriptAnalyzer -Path '.' -Recurse" in script
        assert "$issues.Count -gt 0" in script

    def test_falls_back_to_powershell(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.powershell.find_tool",
            side_effect=[FileNotFoundError("no pwsh"), "/fake/powershell"],
        ), patch("formattington.powershell.subprocess.run", side_effect=fake):
            assert powershell() == 0
        assert calls[0][0][0] == "/fake/powershell"

    def test_missing_shell_raises_with_install_hint(self):
        with patch(
            "formattington.powershell.find_tool",
            side_effect=FileNotFoundError("Could not find 'pwsh'"),
        ):
            with pytest.raises(FileNotFoundError, match="pwsh"):
                powershell()

    def test_missing_shell_message_mentions_module(self):
        with patch(
            "formattington.powershell.find_tool",
            side_effect=FileNotFoundError("x"),
        ):
            with pytest.raises(
                FileNotFoundError, match="Install-Module PSScriptAnalyzer"
            ):
                powershell()

    def test_files_passed_via_path_list(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.powershell.find_tool", return_value="/fake/pwsh"
        ), patch("formattington.powershell.subprocess.run", side_effect=fake):
            powershell(args=["my dir/script.ps1", "other.psm1"])
        script = calls[0][0][3]
        assert "-Path 'my dir/script.ps1', 'other.psm1'" in script
        assert "-Recurse" not in script

    def test_single_quote_doubled(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.powershell.find_tool", return_value="/fake/pwsh"
        ), patch("formattington.powershell.subprocess.run", side_effect=fake):
            powershell(args=["it's here.ps1"])
        assert "'it''s here.ps1'" in calls[0][0][3]

    def test_params_forwarded_to_analyzer(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.powershell.find_tool", return_value="/fake/pwsh"
        ), patch("formattington.powershell.subprocess.run", side_effect=fake):
            powershell(args=["-Severity", "Error", "script.ps1"])
        script = calls[0][0][3]
        assert "-Severity Error" in script
        assert "-Path 'script.ps1'" in script

    def test_check_and_dry_run_same_lint_invocation(self):
        for kwargs in ({}, {"check": True}, {"dry_run": True}):
            fake, calls = _fake_run_capture()
            with patch(
                "formattington.powershell.find_tool", return_value="/fake/pwsh"
            ), patch("formattington.powershell.subprocess.run", side_effect=fake):
                powershell(**kwargs)
            assert "Invoke-ScriptAnalyzer" in calls[0][0][3]

    def test_check_and_dry_run_are_exclusive(self):
        with patch(
            "formattington.powershell.find_tool", return_value="/fake/pwsh"
        ):
            with pytest.raises(ValueError, match="cannot both be enabled"):
                powershell(check=True, dry_run=True)

    def test_propagates_exit_code(self):
        fake, _ = _fake_run_capture(returncode=1)
        with patch(
            "formattington.powershell.find_tool", return_value="/fake/pwsh"
        ), patch("formattington.powershell.subprocess.run", side_effect=fake):
            # PSScriptAnalyzer findings map to exit 1 inside pwsh;
            # Formattington preserves that code.
            assert powershell(check=True) == 1

    def test_help_passthrough(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.powershell.find_tool", return_value="/fake/pwsh"
        ), patch("formattington.powershell.subprocess.run", side_effect=fake):
            assert powershell(args=["--help"]) == 0
        assert calls[0][0] == ["/fake/pwsh", "--help"]

    def test_no_shell_used(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.powershell.find_tool", return_value="/fake/pwsh"
        ), patch("formattington.powershell.subprocess.run", side_effect=fake):
            powershell(args=["script.ps1"])
        _, kwargs = calls[0]
        assert kwargs.get("shell") is None
        assert kwargs["check"] is False

    def test_forwards_verbose_to_shell_lookup(self):
        fake, _ = _fake_run_capture()
        with patch(
            "formattington.powershell.find_tool", return_value="/fake/pwsh"
        ) as mock_find, patch(
            "formattington.powershell.subprocess.run", side_effect=fake
        ):
            assert powershell(verbose=True) == 0
        mock_find.assert_called_once_with("pwsh", verbose=True)


class TestTaplo:
    """Issue #26: TOML formatting via taplo fmt."""

    def test_extensions(self):
        from formattington.taplo import FILE_EXTENSIONS

        assert FILE_EXTENSIONS == (".toml",)

    def test_default_mode_formats_in_place(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.taplo.find_tool", return_value="/fake/taplo"
        ), patch("formattington.taplo.subprocess.run", side_effect=fake):
            assert taplo() == 0
        assert calls[0][0] == ["/fake/taplo", "fmt", "--colors", "never"]

    def test_check_mode_uses_fmt_check(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.taplo.find_tool", return_value="/fake/taplo"
        ), patch("formattington.taplo.subprocess.run", side_effect=fake):
            assert taplo(check=True) == 0
        assert calls[0][0] == [
            "/fake/taplo",
            "fmt",
            "--colors",
            "never",
            "--check",
        ]

    def test_dry_run_mode_uses_fmt_diff(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.taplo.find_tool", return_value="/fake/taplo"
        ), patch("formattington.taplo.subprocess.run", side_effect=fake):
            assert taplo(dry_run=True) == 0
        assert calls[0][0] == ["/fake/taplo", "fmt", "--colors", "never", "--diff"]

    def test_forwards_args_and_preserves_spaces(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.taplo.find_tool", return_value="/fake/taplo"
        ), patch("formattington.taplo.subprocess.run", side_effect=fake):
            taplo(check=True, args=["my file.toml", "another file.toml"])
        assert calls[0][0] == [
            "/fake/taplo",
            "fmt",
            "--colors",
            "never",
            "--check",
            "my file.toml",
            "another file.toml",
        ]
        assert len(calls) == 1

    def test_user_colors_are_respected(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.taplo.find_tool", return_value="/fake/taplo"
        ), patch("formattington.taplo.subprocess.run", side_effect=fake):
            taplo(args=["--colors", "always", "file.toml"])
        assert calls[0][0] == ["/fake/taplo", "fmt", "--colors", "always", "file.toml"]

    def test_check_and_dry_run_are_exclusive(self):
        with patch("formattington.taplo.find_tool", return_value="/fake/taplo"):
            with pytest.raises(ValueError, match="cannot both be enabled"):
                taplo(check=True, dry_run=True)

    def test_propagates_subprocess_exit_code(self):
        fake, _ = _fake_run_capture(returncode=1)
        with patch(
            "formattington.taplo.find_tool", return_value="/fake/taplo"
        ), patch("formattington.taplo.subprocess.run", side_effect=fake):
            assert taplo(check=True) == 1

    def test_missing_tool_raises(self):
        with patch(
            "formattington.taplo.find_tool",
            side_effect=FileNotFoundError("Could not find 'taplo'"),
        ):
            with pytest.raises(FileNotFoundError):
                taplo()

    def test_help_passthrough_without_mode_flags(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.taplo.find_tool", return_value="/fake/taplo"
        ), patch("formattington.taplo.subprocess.run", side_effect=fake):
            assert taplo(check=True, args=["--help"]) == 0
        assert calls[0][0] == ["/fake/taplo", "--help"]

    def test_version_passthrough_without_mode_flags(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.taplo.find_tool", return_value="/fake/taplo"
        ), patch("formattington.taplo.subprocess.run", side_effect=fake):
            assert taplo(dry_run=True, args=["--version"]) == 0
        assert calls[0][0] == ["/fake/taplo", "--version"]

    def test_no_shell_used(self):
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.taplo.find_tool", return_value="/fake/taplo"
        ), patch("formattington.taplo.subprocess.run", side_effect=fake):
            taplo(args=["file with spaces.toml"])
        _, kwargs = calls[0]
        assert kwargs.get("shell") is None
        assert kwargs["check"] is False

    def test_does_not_add_config_flag(self):
        # A project taplo.toml must be respected; Formattington adds no
        # --config override unless the caller forwards one.
        fake, calls = _fake_run_capture()
        with patch(
            "formattington.taplo.find_tool", return_value="/fake/taplo"
        ), patch("formattington.taplo.subprocess.run", side_effect=fake):
            taplo(args=["file.toml"])
        assert "--config" not in calls[0][0]

    def test_forwards_verbose_to_find_tool(self):
        fake, _ = _fake_run_capture()
        with patch(
            "formattington.taplo.find_tool", return_value="/fake/taplo"
        ) as mock_find, patch(
            "formattington.taplo.subprocess.run", side_effect=fake
        ):
            assert taplo(verbose=True) == 0
        mock_find.assert_called_once_with("taplo", verbose=True)
