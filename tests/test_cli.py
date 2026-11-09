"""Tests for CLI parsing, formatter selection, and forwarding."""

from __future__ import annotations

import sys
from unittest.mock import MagicMock, patch

import pytest

import formattington.cli as cli
from formattington.cli import main


@pytest.fixture
def mock_formatters(monkeypatch):
    """Replace FORMATTERS with mocks and restore after.

    Returns a dict of mocks keyed by formatter name.
    """
    mocks = {
        name: MagicMock(return_value=0) for name in ("shfmt", "clang-format", "black")
    }
    original = dict(cli.FORMATTERS)
    cli.FORMATTERS.update(mocks)
    try:
        yield mocks
    finally:
        cli.FORMATTERS.clear()
        cli.FORMATTERS.update(original)


def _set_argv(monkeypatch, argv):
    monkeypatch.setattr(sys, "argv", argv)


class TestFormatterSelection:
    def test_shfmt_selection(self, monkeypatch, mock_formatters):
        _set_argv(monkeypatch, ["formattington", "shfmt"])
        assert main() == 0
        mock_formatters["shfmt"].assert_called_once_with(
            check=False, dry_run=False, args=[]
        )
        mock_formatters["clang-format"].assert_not_called()

    def test_clang_format_selection(self, monkeypatch, mock_formatters):
        _set_argv(monkeypatch, ["formattington", "clang-format"])
        assert main() == 0
        mock_formatters["clang-format"].assert_called_once_with(
            check=False, dry_run=False, args=[]
        )
        mock_formatters["shfmt"].assert_not_called()

    def test_black_selection(self, monkeypatch, mock_formatters):
        _set_argv(monkeypatch, ["formattington", "black"])
        assert main() == 0
        mock_formatters["black"].assert_called_once_with(
            check=False, dry_run=False, args=[]
        )
        mock_formatters["shfmt"].assert_not_called()

    def test_python_alias_selection(self, monkeypatch, mock_formatters):
        _set_argv(monkeypatch, ["formattington", "python"])
        assert main() == 0
        mock_formatters["black"].assert_called_once_with(
            check=False, dry_run=False, args=[]
        )

    def test_all_runs_black_exactly_once(self, monkeypatch, mock_formatters):
        _set_argv(monkeypatch, ["formattington", "all"])
        assert main() == 0
        mock_formatters["black"].assert_called_once_with(
            check=False, dry_run=False, args=[]
        )

    def test_all_runs_every_formatter(self, monkeypatch, mock_formatters):
        _set_argv(monkeypatch, ["formattington", "all"])
        assert main() == 0
        for mock in mock_formatters.values():
            mock.assert_called_once_with(check=False, dry_run=False, args=[])

    def test_all_preserves_last_nonzero_exit(self, monkeypatch, mock_formatters):
        mock_formatters["shfmt"].return_value = 5
        mock_formatters["clang-format"].return_value = 0
        _set_argv(monkeypatch, ["formattington", "all"])
        assert main() == 5

    def test_all_with_extra_args_exits_2(self, monkeypatch, mock_formatters):
        _set_argv(monkeypatch, ["formattington", "all", "extra"])
        with pytest.raises(SystemExit) as exc_info:
            main()
        assert exc_info.value.code == 2

    def test_no_formatter_is_a_noop(self, monkeypatch, mock_formatters):
        _set_argv(monkeypatch, ["formattington"])
        assert main() == 0
        for mock in mock_formatters.values():
            mock.assert_not_called()


class TestModes:
    def test_check_before_formatter(self, monkeypatch, mock_formatters):
        _set_argv(monkeypatch, ["formattington", "--check", "shfmt"])
        assert main() == 0
        mock_formatters["shfmt"].assert_called_once_with(
            check=True, dry_run=False, args=[]
        )

    def test_check_before_black(self, monkeypatch, mock_formatters):
        _set_argv(monkeypatch, ["formattington", "--check", "black"])
        assert main() == 0
        mock_formatters["black"].assert_called_once_with(
            check=True, dry_run=False, args=[]
        )

    def test_dry_run_before_formatter(self, monkeypatch, mock_formatters):
        _set_argv(monkeypatch, ["formattington", "--dry-run", "clang-format"])
        assert main() == 0
        mock_formatters["clang-format"].assert_called_once_with(
            check=False, dry_run=True, args=[]
        )

    def test_dry_run_before_black(self, monkeypatch, mock_formatters):
        _set_argv(monkeypatch, ["formattington", "--dry-run", "black"])
        assert main() == 0
        mock_formatters["black"].assert_called_once_with(
            check=False, dry_run=True, args=[]
        )

    def test_check_propagates_to_all(self, monkeypatch, mock_formatters):
        _set_argv(monkeypatch, ["formattington", "--check", "all"])
        assert main() == 0
        for mock in mock_formatters.values():
            mock.assert_called_once_with(check=True, dry_run=False, args=[])

    def test_check_and_dry_run_are_mutually_exclusive(self, monkeypatch, mock_formatters):
        _set_argv(monkeypatch, ["formattington", "--check", "--dry-run", "shfmt"])
        with pytest.raises(SystemExit) as exc_info:
            main()
        assert exc_info.value.code == 2

    def test_unknown_global_flag_exits_2(self, monkeypatch, mock_formatters):
        _set_argv(monkeypatch, ["formattington", "--bogus", "shfmt"])
        with pytest.raises(SystemExit) as exc_info:
            main()
        assert exc_info.value.code == 2


class TestUnknownWords:
    @pytest.mark.parametrize("word", ["foobar", "unknown-word", "prettier", "c"])
    def test_unknown_word_exits_2(self, monkeypatch, mock_formatters, word):
        _set_argv(monkeypatch, ["formattington", word])
        with pytest.raises(SystemExit) as exc_info:
            main()
        assert exc_info.value.code == 2
        for mock in mock_formatters.values():
            mock.assert_not_called()


class TestVerbose:
    def test_verbose_before_formatter_passed_through(
        self, monkeypatch, mock_formatters
    ):
        _set_argv(monkeypatch, ["formattington", "--verbose", "shfmt"])
        assert main() == 0
        mock_formatters["shfmt"].assert_called_once_with(
            check=False, dry_run=False, args=[], verbose=True
        )

    def test_verbose_before_black(self, monkeypatch, mock_formatters):
        _set_argv(monkeypatch, ["formattington", "--verbose", "black"])
        assert main() == 0
        mock_formatters["black"].assert_called_once_with(
            check=False, dry_run=False, args=[], verbose=True
        )

    def test_verbose_with_check_and_all(self, monkeypatch, mock_formatters):
        _set_argv(monkeypatch, ["formattington", "--check", "--verbose", "all"])
        assert main() == 0
        for mock in mock_formatters.values():
            mock.assert_called_once_with(
                check=True, dry_run=False, args=[], verbose=True
            )

    def test_help_lists_verbose(self, monkeypatch, capsys):
        _set_argv(monkeypatch, ["formattington", "--help"])
        with pytest.raises(SystemExit) as exc_info:
            main()
        assert exc_info.value.code == 0
        assert "--verbose" in capsys.readouterr().out


class TestHelpAndVersion:
    def test_help_lists_options(self, monkeypatch, capsys):
        _set_argv(monkeypatch, ["formattington", "--help"])
        with pytest.raises(SystemExit) as exc_info:
            main()
        assert exc_info.value.code == 0
        out = capsys.readouterr().out
        for needle in ("--help", "--version", "--check", "--dry-run"):
            assert needle in out

    def test_help_short_flag(self, monkeypatch, capsys):
        _set_argv(monkeypatch, ["formattington", "-h"])
        with pytest.raises(SystemExit) as exc_info:
            main()
        assert exc_info.value.code == 0

    def test_version_before_formatter(self, monkeypatch, mock_formatters, capsys):
        _set_argv(monkeypatch, ["formattington", "--version"])
        assert main() == 0
        assert "0.1.0" in capsys.readouterr().out
        for mock in mock_formatters.values():
            mock.assert_not_called()

    def test_version_before_formatter_does_not_run_formatter(
        self, monkeypatch, mock_formatters
    ):
        _set_argv(monkeypatch, ["formattington", "--version", "shfmt"])
        assert main() == 0
        for mock in mock_formatters.values():
            mock.assert_not_called()


class TestArgForwarding:
    def test_forward_args_after_shfmt(self, monkeypatch, mock_formatters):
        _set_argv(monkeypatch, ["formattington", "shfmt", "--keep-padding", "-i", "2"])
        assert main() == 0
        mock_formatters["shfmt"].assert_called_once_with(
            check=False, dry_run=False, args=["--keep-padding", "-i", "2"]
        )

    def test_forward_args_after_clang_format(self, monkeypatch, mock_formatters):
        _set_argv(monkeypatch, ["formattington", "clang-format", "--style=file"])
        assert main() == 0
        mock_formatters["clang-format"].assert_called_once_with(
            check=False, dry_run=False, args=["--style=file"]
        )

    def test_forward_args_after_black(self, monkeypatch, mock_formatters):
        _set_argv(monkeypatch, ["formattington", "black", "--line-length", "88"])
        assert main() == 0
        mock_formatters["black"].assert_called_once_with(
            check=False, dry_run=False, args=["--line-length", "88"]
        )

    def test_forward_preserves_spaces_in_args(self, monkeypatch, mock_formatters):
        _set_argv(
            monkeypatch,
            ["formattington", "shfmt", "file with spaces.sh", "-i", "2"],
        )
        assert main() == 0
        mock_formatters["shfmt"].assert_called_once_with(
            check=False,
            dry_run=False,
            args=["file with spaces.sh", "-i", "2"],
        )

    def test_formatter_args_are_not_parsed_by_formattington(
        self, monkeypatch, mock_formatters
    ):
        # "--check" after the formatter belongs to the formatter, not to
        # Formattington's own global modes.
        _set_argv(monkeypatch, ["formattington", "shfmt", "--check"])
        assert main() == 0
        mock_formatters["shfmt"].assert_called_once_with(
            check=False, dry_run=False, args=["--check"]
        )


class TestMissingToolExit127:
    def test_single_formatter_missing_returns_127(self, monkeypatch, capsys):
        _set_argv(monkeypatch, ["formattington", "shfmt"])
        with patch(
            "formattington.shfmt.find_tool",
            side_effect=FileNotFoundError("Could not find 'shfmt'"),
        ):
            assert main() == 127
        assert "Could not find 'shfmt'" in capsys.readouterr().err

    def test_clang_format_missing_returns_127(self, monkeypatch, capsys):
        _set_argv(monkeypatch, ["formattington", "clang-format"])
        with patch(
            "formattington.clang_format.find_tool",
            side_effect=FileNotFoundError("Could not find 'clang-format'"),
        ):
            assert main() == 127

    def test_black_missing_returns_127(self, monkeypatch, capsys):
        _set_argv(monkeypatch, ["formattington", "black"])
        with patch(
            "formattington.format_python.find_tool",
            side_effect=FileNotFoundError("Could not find 'black'"),
        ):
            assert main() == 127

    def test_all_returns_127_when_any_missing(self, monkeypatch):
        # shfmt missing, clang-format and black present: still runs the
        # present ones, final exit is 127.
        _set_argv(monkeypatch, ["formattington", "all"])
        # shfmt raises via find_tool; clang-format and black succeed via
        # mocked subprocess.run so no real binary is needed.
        def _fake_run(cmd, **kwargs):
            result = MagicMock()
            result.returncode = 0
            return result

        with patch(
            "formattington.shfmt.find_tool",
            side_effect=FileNotFoundError("Could not find 'shfmt'"),
        ), patch(
            "formattington.clang_format.find_tool", return_value="/fake/clang-format"
        ), patch(
            "formattington.clang_format.subprocess.run", side_effect=_fake_run
        ), patch(
            "formattington.format_python.find_tool", return_value="/fake/black"
        ), patch(
            "formattington.format_python.subprocess.run", side_effect=_fake_run
        ):
            assert main() == 127

    def test_unsupported_platform_returns_127(self, monkeypatch):
        _set_argv(monkeypatch, ["formattington", "shfmt"])
        with patch(
            "formattington.shfmt.find_tool",
            side_effect=RuntimeError("Unsupported platform: freebsd-sparc"),
        ):
            assert main() == 127

    @pytest.mark.parametrize(
        "mode_args",
        [[], ["--check"], ["--dry-run"]],
        ids=["default", "check", "dry-run"],
    )
    @pytest.mark.parametrize("formatter", ["shfmt", "clang-format", "black"])
    def test_missing_tool_returns_127_in_every_mode(
        self, monkeypatch, capsys, mode_args, formatter
    ):
        # Missing-executable handling is identical across default,
        # --check, and --dry-run: no traceback, stderr via
        # "formattington: ...", exit 127.
        _set_argv(monkeypatch, ["formattington", *mode_args, formatter])
        patch_target = {
            "shfmt": "formattington.shfmt.find_tool",
            "clang-format": "formattington.clang_format.find_tool",
            "black": "formattington.format_python.find_tool",
        }[formatter]
        with patch(
            patch_target,
            side_effect=FileNotFoundError(
                f"Could not find '{formatter}'. Expected a bundled executable"
            ),
        ):
            assert main() == 127
        err = capsys.readouterr().err
        assert err.startswith("formattington:")
        assert f"Could not find '{formatter}'" in err
        assert "Traceback" not in err

    def test_missing_tool_with_forwarded_args_returns_127(self, monkeypatch, capsys):
        # "--check" after the formatter is forwarded to the formatter,
        # but a missing executable still exits 127 before any run.
        _set_argv(monkeypatch, ["formattington", "shfmt", "--check"])
        with patch(
            "formattington.shfmt.find_tool",
            side_effect=FileNotFoundError("Could not find 'shfmt'"),
        ):
            assert main() == 127
        assert "Could not find 'shfmt'" in capsys.readouterr().err

    def test_help_and_version_do_not_require_tools(self, monkeypatch):
        # --help and --version must work even when tools are missing.
        _set_argv(monkeypatch, ["formattington", "--help"])
        with pytest.raises(SystemExit) as exc_info:
            main()
        assert exc_info.value.code == 0
        _set_argv(monkeypatch, ["formattington", "--version"])
        assert main() == 0

    def test_all_returns_127_when_all_missing(self, monkeypatch, capsys):
        _set_argv(monkeypatch, ["formattington", "--check", "all"])
        with patch(
            "formattington.shfmt.find_tool",
            side_effect=FileNotFoundError("Could not find 'shfmt'"),
        ), patch(
            "formattington.clang_format.find_tool",
            side_effect=FileNotFoundError("Could not find 'clang-format'"),
        ), patch(
            "formattington.format_python.find_tool",
            side_effect=FileNotFoundError("Could not find 'black'"),
        ):
            assert main() == 127
        err = capsys.readouterr().err
        assert err.count("formattington:") == 3
