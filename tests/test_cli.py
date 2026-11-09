"""Tests for CLI parsing, formatter selection, and forwarding."""

from __future__ import annotations

from unittest.mock import MagicMock, patch

import pytest

import formattington.cli as cli
from formattington.cli import main
from tests.conftest import set_argv


class TestFormatterSelection:
    @pytest.mark.parametrize(
        "formatter",
        ["shfmt", "clang-format", "black", "prettier", "powershell", "taplo"],
    )
    def test_single_tool_selection(self, monkeypatch, mock_formatters, formatter):
        set_argv(monkeypatch, ["formattington", formatter])
        assert main() == 0
        mock_formatters[formatter].assert_called_once_with(
            check=False, dry_run=False, args=[], verbose=False
        )
        for name, mock in mock_formatters.items():
            if name != formatter:
                mock.assert_not_called()

    def test_python_alias_selection(self, monkeypatch, mock_formatters):
        set_argv(monkeypatch, ["formattington", "python"])
        assert main() == 0
        mock_formatters["black"].assert_called_once_with(
            check=False, dry_run=False, args=[], verbose=False
        )

    def test_md_alias_selection(self, monkeypatch, mock_formatters):
        """Issue #6: 'md' resolves to the existing Prettier integration."""
        from formattington.cli import LANGUAGE_ALIASES

        assert LANGUAGE_ALIASES["md"] == "prettier"
        set_argv(monkeypatch, ["formattington", "md"])
        assert main() == 0
        mock_formatters["prettier"].assert_called_once_with(
            check=False, dry_run=False, args=[], verbose=False
        )
        mock_formatters["black"].assert_not_called()

    def test_markdown_alias_selection(self, monkeypatch, mock_formatters):
        """Issue #6: 'markdown' resolves to the existing Prettier integration."""
        from formattington.cli import LANGUAGE_ALIASES

        assert LANGUAGE_ALIASES["markdown"] == "prettier"
        set_argv(monkeypatch, ["formattington", "markdown"])
        assert main() == 0
        mock_formatters["prettier"].assert_called_once_with(
            check=False, dry_run=False, args=[], verbose=False
        )
        mock_formatters["black"].assert_not_called()

    def test_markdown_alias_check_and_dry_run(self, monkeypatch, mock_formatters):
        """Aliases support the same modes as 'prettier' (check/dry-run)."""
        set_argv(monkeypatch, ["formattington", "--check", "md"])
        assert main() == 0
        mock_formatters["prettier"].assert_called_once_with(
            check=True, dry_run=False, args=[], verbose=False
        )
        set_argv(monkeypatch, ["formattington", "--dry-run", "markdown"])
        assert main() == 0
        mock_formatters["prettier"].assert_called_with(
            check=False, dry_run=True, args=[], verbose=False
        )

    def test_markdown_alias_forwards_args(self, monkeypatch, mock_formatters):
        """Forwarded Prettier args (including native --check) pass through."""
        set_argv(
            monkeypatch,
            ["formattington", "markdown", "--single-quote", "my file.md"],
        )
        assert main() == 0
        mock_formatters["prettier"].assert_called_once_with(
            check=False, dry_run=False, args=["--single-quote", "my file.md"], verbose=False
        )
        set_argv(monkeypatch, ["formattington", "md", "--check"])
        assert main() == 0
        mock_formatters["prettier"].assert_called_with(
            check=False, dry_run=False, args=["--check"], verbose=False
        )

    def test_markdown_alias_missing_prettier_returns_127(
        self, monkeypatch, capsys
    ):
        """Missing Prettier via an alias exits 127 with no traceback."""
        set_argv(monkeypatch, ["formattington", "md"])
        with patch(
            "formattington.prettier.find_tool",
            side_effect=FileNotFoundError("Could not find 'prettier'"),
        ):
            assert main() == 127
        err = capsys.readouterr().err
        assert err.startswith("formattington:")
        assert "Could not find 'prettier'" in err
        assert "Traceback" not in err

    def test_all_runs_black_exactly_once(self, monkeypatch, mock_formatters):
        set_argv(monkeypatch, ["formattington", "all"])
        assert main() == 0
        mock_formatters["black"].assert_called_once_with(
            check=False, dry_run=False, args=[], verbose=False
        )

    def test_all_runs_prettier_exactly_once(self, monkeypatch, mock_formatters):
        set_argv(monkeypatch, ["formattington", "all"])
        assert main() == 0
        mock_formatters["prettier"].assert_called_once_with(
            check=False, dry_run=False, args=[], verbose=False
        )


class TestPowershellSelection:
    """Issue #25: `powershell` tool and `ps1` alias."""

    def test_ps1_alias_selection(self, monkeypatch, mock_formatters):
        from formattington.cli import LANGUAGE_ALIASES

        assert LANGUAGE_ALIASES["ps1"] == "powershell"
        set_argv(monkeypatch, ["formattington", "ps1"])
        assert main() == 0
        mock_formatters["powershell"].assert_called_once_with(
            check=False, dry_run=False, args=[], verbose=False
        )
        mock_formatters["black"].assert_not_called()

    def test_ps1_alias_modes_and_forwarding(self, monkeypatch, mock_formatters):
        set_argv(monkeypatch, ["formattington", "--check", "ps1"])
        assert main() == 0
        mock_formatters["powershell"].assert_called_once_with(
            check=True, dry_run=False, args=[], verbose=False
        )
        set_argv(monkeypatch, ["formattington", "ps1", "-Severity", "Error"])
        assert main() == 0
        mock_formatters["powershell"].assert_called_with(
            check=False, dry_run=False, args=["-Severity", "Error"], verbose=False
        )

    def test_powershell_missing_returns_127(self, monkeypatch, capsys):
        set_argv(monkeypatch, ["formattington", "ps1"])
        with patch(
            "formattington.powershell.find_tool",
            side_effect=FileNotFoundError("Could not find 'pwsh' or 'powershell'"),
        ):
            assert main() == 127
        err = capsys.readouterr().err
        assert err.startswith("formattington:")
        assert "Traceback" not in err

    def test_help_mentions_powershell_alias(self, monkeypatch, capsys):
        set_argv(monkeypatch, ["formattington", "--help"])
        with pytest.raises(SystemExit) as exc_info:
            main()
        assert exc_info.value.code == 0
        out = capsys.readouterr().out
        assert "ps1" in out
        assert "powershell" in out
        assert "PSScriptAnalyzer" in out


class TestTaploSelection:
    """Issue #26: `taplo` tool and `toml` alias."""

    def test_toml_alias_selection(self, monkeypatch, mock_formatters):
        from formattington.cli import LANGUAGE_ALIASES

        assert LANGUAGE_ALIASES["toml"] == "taplo"
        set_argv(monkeypatch, ["formattington", "toml"])
        assert main() == 0
        mock_formatters["taplo"].assert_called_once_with(
            check=False, dry_run=False, args=[], verbose=False
        )
        mock_formatters["black"].assert_not_called()

    def test_toml_alias_modes_and_forwarding(self, monkeypatch, mock_formatters):
        set_argv(monkeypatch, ["formattington", "--check", "toml"])
        assert main() == 0
        mock_formatters["taplo"].assert_called_once_with(
            check=True, dry_run=False, args=[], verbose=False
        )
        set_argv(monkeypatch, ["formattington", "toml", "--option", "align_comments=true"])
        assert main() == 0
        mock_formatters["taplo"].assert_called_with(
            check=False, dry_run=False, args=["--option", "align_comments=true"], verbose=False
        )

    def test_taplo_missing_returns_127(self, monkeypatch, capsys):
        set_argv(monkeypatch, ["formattington", "taplo"])
        with patch(
            "formattington.taplo.find_tool",
            side_effect=FileNotFoundError("Could not find 'taplo'"),
        ):
            assert main() == 127
        err = capsys.readouterr().err
        assert err.startswith("formattington:")
        assert "Traceback" not in err

    def test_help_mentions_toml_alias(self, monkeypatch, capsys):
        set_argv(monkeypatch, ["formattington", "--help"])
        with pytest.raises(SystemExit) as exc_info:
            main()
        assert exc_info.value.code == 0
        out = capsys.readouterr().out
        assert "toml" in out
        assert "taplo" in out

    def test_all_runs_every_formatter(self, monkeypatch, mock_formatters):
        set_argv(monkeypatch, ["formattington", "all"])
        assert main() == 0
        for mock in mock_formatters.values():
            mock.assert_called_once_with(check=False, dry_run=False, args=[], verbose=False)

    def test_all_preserves_last_nonzero_exit(self, monkeypatch, mock_formatters):
        mock_formatters["shfmt"].return_value = 5
        mock_formatters["clang-format"].return_value = 0
        set_argv(monkeypatch, ["formattington", "all"])
        assert main() == 5

    def test_all_with_extra_args_exits_2(self, monkeypatch, mock_formatters):
        set_argv(monkeypatch, ["formattington", "all", "extra"])
        with pytest.raises(SystemExit) as exc_info:
            main()
        assert exc_info.value.code == 2

    def test_no_formatter_is_a_noop(self, monkeypatch, mock_formatters):
        set_argv(monkeypatch, ["formattington"])
        assert main() == 0
        for mock in mock_formatters.values():
            mock.assert_not_called()


class TestModes:
    @pytest.mark.parametrize(
        "formatter",
        ["shfmt", "clang-format", "black", "prettier", "powershell", "taplo"],
    )
    @pytest.mark.parametrize(
        ("flag", "expected_check", "expected_dry_run"),
        [("--check", True, False), ("--dry-run", False, True)],
        ids=["check", "dry-run"],
    )
    def test_check_dry_run_before_formatter(
        self, monkeypatch, mock_formatters, formatter, flag, expected_check, expected_dry_run
    ):
        set_argv(monkeypatch, ["formattington", flag, formatter])
        assert main() == 0
        mock_formatters[formatter].assert_called_once_with(
            check=expected_check, dry_run=expected_dry_run, args=[], verbose=False
        )

    def test_check_propagates_to_all(self, monkeypatch, mock_formatters):
        set_argv(monkeypatch, ["formattington", "--check", "all"])
        assert main() == 0
        for mock in mock_formatters.values():
            mock.assert_called_once_with(check=True, dry_run=False, args=[], verbose=False)

    def test_check_and_dry_run_are_mutually_exclusive(self, monkeypatch, mock_formatters):
        set_argv(monkeypatch, ["formattington", "--check", "--dry-run", "shfmt"])
        with pytest.raises(SystemExit) as exc_info:
            main()
        assert exc_info.value.code == 2

    def test_unknown_global_flag_exits_2(self, monkeypatch, mock_formatters):
        set_argv(monkeypatch, ["formattington", "--bogus", "shfmt"])
        with pytest.raises(SystemExit) as exc_info:
            main()
        assert exc_info.value.code == 2


class TestUnknownWords:
    @pytest.mark.parametrize("word", ["foobar", "unknown-word", "cs"])
    def test_unknown_word_exits_2(self, monkeypatch, mock_formatters, word):
        set_argv(monkeypatch, ["formattington", word])
        with pytest.raises(SystemExit) as exc_info:
            main()
        assert exc_info.value.code == 2
        for mock in mock_formatters.values():
            mock.assert_not_called()


class TestLanguageAliases:
    """Issue #2: language / file-type alias selection.

    Every alias in the README (c, cpp, c++, bash, md, markdown, html)
    plus python/py resolves to a registered formatter with the same
    --check/--dry-run and forwarding semantics as formatter selection.
    """

    ALIAS_TO_FORMATTER = {
        "c": "clang-format",
        "cpp": "clang-format",
        "c++": "clang-format",
        "bash": "shfmt",
        "python": "black",
        "py": "black",
        "md": "prettier",
        "markdown": "prettier",
        "html": "prettier",
    }

    @pytest.mark.parametrize(
        ("alias", "formatter"),
        sorted(ALIAS_TO_FORMATTER.items()),
        ids=sorted(ALIAS_TO_FORMATTER),
    )
    def test_alias_registry(self, alias, formatter):
        assert cli.LANGUAGE_ALIASES[alias] == formatter
        assert formatter in cli.FORMATTERS

    @pytest.mark.parametrize(
        ("alias", "formatter"),
        sorted(ALIAS_TO_FORMATTER.items()),
        ids=sorted(ALIAS_TO_FORMATTER),
    )
    def test_alias_dispatches_to_formatter(
        self, monkeypatch, mock_formatters, alias, formatter
    ):
        set_argv(monkeypatch, ["formattington", alias])
        assert main() == 0
        mock_formatters[formatter].assert_called_once_with(
            check=False, dry_run=False, args=[], verbose=False
        )
        for name, mock in mock_formatters.items():
            if name != formatter:
                mock.assert_not_called()

    def test_alias_check_and_dry_run(self, monkeypatch, mock_formatters):
        set_argv(monkeypatch, ["formattington", "--check", "c"])
        assert main() == 0
        mock_formatters["clang-format"].assert_called_once_with(
            check=True, dry_run=False, args=[], verbose=False
        )
        set_argv(monkeypatch, ["formattington", "--dry-run", "bash"])
        assert main() == 0
        mock_formatters["shfmt"].assert_called_with(
            check=False, dry_run=True, args=[], verbose=False
        )

    def test_alias_forwards_args(self, monkeypatch, mock_formatters):
        set_argv(monkeypatch, ["formattington", "cpp", "--style=file"])
        assert main() == 0
        mock_formatters["clang-format"].assert_called_once_with(
            check=False, dry_run=False, args=["--style=file"], verbose=False
        )

    def test_alias_forwarded_check_is_native(self, monkeypatch, mock_formatters):
        set_argv(monkeypatch, ["formattington", "html", "--check"])
        assert main() == 0
        mock_formatters["prettier"].assert_called_once_with(
            check=False, dry_run=False, args=["--check"], verbose=False
        )

    def test_alias_verbose(self, monkeypatch, mock_formatters):
        set_argv(monkeypatch, ["formattington", "--verbose", "bash"])
        assert main() == 0
        mock_formatters["shfmt"].assert_called_once_with(
            check=False, dry_run=False, args=[], verbose=True
        )

    def test_alias_missing_tool_exits_127(self, monkeypatch, capsys):
        set_argv(monkeypatch, ["formattington", "c"])
        with patch(
            "formattington.clang_format.find_tool",
            side_effect=FileNotFoundError("Could not find 'clang-format'"),
        ):
            assert main() == 127
        err = capsys.readouterr().err
        assert err.startswith("formattington:")
        assert "Traceback" not in err

    def test_unregistered_alias_is_not_yet_supported(
        self, monkeypatch, mock_formatters, capsys
    ):
        monkeypatch.setitem(cli.LANGUAGE_ALIASES, "cobol", "cobol-formatter")
        set_argv(monkeypatch, ["formattington", "cobol"])
        with pytest.raises(SystemExit) as exc_info:
            main()
        assert exc_info.value.code == 2
        assert "not yet supported" in capsys.readouterr().err
        for mock in mock_formatters.values():
            mock.assert_not_called()

    def test_help_lists_all_aliases(self, monkeypatch, capsys):
        set_argv(monkeypatch, ["formattington", "--help"])
        with pytest.raises(SystemExit) as exc_info:
            main()
        assert exc_info.value.code == 0
        out = capsys.readouterr().out
        for needle in ("c/cpp/c++", "bash", "python/py", "html", "clang-format"):
            assert needle in out

    def test_all_aliases_have_backends(self):
        assert all(
            target in cli.FORMATTERS
            for target in cli.LANGUAGE_ALIASES.values()
        )


class TestVerbose:
    @pytest.mark.parametrize(
        "formatter",
        ["shfmt", "clang-format", "black", "prettier", "powershell", "taplo"],
    )
    def test_verbose_before_formatter(
        self, monkeypatch, mock_formatters, formatter
    ):
        set_argv(monkeypatch, ["formattington", "--verbose", formatter])
        assert main() == 0
        mock_formatters[formatter].assert_called_once_with(
            check=False, dry_run=False, args=[], verbose=True
        )

    def test_verbose_with_check_and_all(self, monkeypatch, mock_formatters):
        set_argv(monkeypatch, ["formattington", "--check", "--verbose", "all"])
        assert main() == 0
        for mock in mock_formatters.values():
            mock.assert_called_once_with(
                check=True, dry_run=False, args=[], verbose=True
            )

    def test_help_lists_verbose(self, monkeypatch, capsys):
        set_argv(monkeypatch, ["formattington", "--help"])
        with pytest.raises(SystemExit) as exc_info:
            main()
        assert exc_info.value.code == 0
        assert "--verbose" in capsys.readouterr().out


class TestHelpAndVersion:
    def test_help_lists_options(self, monkeypatch, capsys):
        set_argv(monkeypatch, ["formattington", "--help"])
        with pytest.raises(SystemExit) as exc_info:
            main()
        assert exc_info.value.code == 0
        out = capsys.readouterr().out
        for needle in ("--help", "--version", "--check", "--dry-run"):
            assert needle in out

    def test_help_describes_markdown_aliases(self, monkeypatch, capsys):
        """Issue #6: --help names Prettier as the Markdown backend."""
        set_argv(monkeypatch, ["formattington", "--help"])
        with pytest.raises(SystemExit) as exc_info:
            main()
        assert exc_info.value.code == 0
        out = capsys.readouterr().out
        assert "md/markdown use prettier" in out
        assert ".md" in out
        assert ".markdown" in out

    def test_help_short_flag(self, monkeypatch, capsys):
        set_argv(monkeypatch, ["formattington", "-h"])
        with pytest.raises(SystemExit) as exc_info:
            main()
        assert exc_info.value.code == 0

    def test_version_before_formatter(self, monkeypatch, mock_formatters, capsys):
        set_argv(monkeypatch, ["formattington", "--version"])
        assert main() == 0
        assert "0.1.0" in capsys.readouterr().out
        for mock in mock_formatters.values():
            mock.assert_not_called()

    def test_version_before_formatter_does_not_run_formatter(
        self, monkeypatch, mock_formatters
    ):
        set_argv(monkeypatch, ["formattington", "--version", "shfmt"])
        assert main() == 0
        for mock in mock_formatters.values():
            mock.assert_not_called()


class TestArgForwarding:
    @pytest.mark.parametrize(
        ("formatter", "forwarded"),
        [
            ("shfmt", ["--keep-padding", "-i", "2"]),
            ("clang-format", ["--style=file"]),
            ("black", ["--line-length", "88"]),
            ("prettier", ["--check"]),
        ],
        ids=["shfmt", "clang-format", "black", "prettier"],
    )
    def test_forward_args_after_formatter(
        self, monkeypatch, mock_formatters, formatter, forwarded
    ):
        set_argv(monkeypatch, ["formattington", formatter, *forwarded])
        assert main() == 0
        mock_formatters[formatter].assert_called_once_with(
            check=False, dry_run=False, args=forwarded, verbose=False
        )

    def test_forward_prettier_config_arg(self, monkeypatch, mock_formatters):
        set_argv(
            monkeypatch, ["formattington", "prettier", "--single-quote", "my file.js"]
        )
        assert main() == 0
        mock_formatters["prettier"].assert_called_once_with(
            check=False, dry_run=False, args=["--single-quote", "my file.js"], verbose=False
        )

    def test_forward_preserves_spaces_in_args(self, monkeypatch, mock_formatters):
        set_argv(
            monkeypatch,
            ["formattington", "shfmt", "file with spaces.sh", "-i", "2"],
        )
        assert main() == 0
        mock_formatters["shfmt"].assert_called_once_with(
            check=False,
            dry_run=False,
            args=["file with spaces.sh", "-i", "2"], verbose=False,
        )

    def test_formatter_args_are_not_parsed_by_formattington(
        self, monkeypatch, mock_formatters
    ):
        # "--check" after the formatter belongs to the formatter, not to
        # Formattington's own global modes.
        set_argv(monkeypatch, ["formattington", "shfmt", "--check"])
        assert main() == 0
        mock_formatters["shfmt"].assert_called_once_with(
            check=False, dry_run=False, args=["--check"], verbose=False
        )


class TestMissingToolExit127:
    def test_all_returns_127_when_any_missing(self, monkeypatch):
        # shfmt missing, clang-format, black, prettier, powershell, and taplo
        # present: still runs the present ones, final exit is 127.
        set_argv(monkeypatch, ["formattington", "all"])
        # shfmt raises via find_tool; the rest succeed via mocked
        # subprocess.run so no real binary is needed.
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
        ), patch(
            "formattington.prettier.find_tool", return_value="/fake/prettier"
        ), patch(
            "formattington.prettier.subprocess.run", side_effect=_fake_run
        ), patch(
            "formattington.powershell.find_tool", return_value="/fake/pwsh"
        ), patch(
            "formattington.powershell.subprocess.run", side_effect=_fake_run
        ), patch(
            "formattington.taplo.find_tool", return_value="/fake/taplo"
        ), patch(
            "formattington.taplo.subprocess.run", side_effect=_fake_run
        ):
            assert main() == 127

    def test_unsupported_platform_returns_127(self, monkeypatch):
        set_argv(monkeypatch, ["formattington", "shfmt"])
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
    @pytest.mark.parametrize(
        "formatter",
        ["shfmt", "clang-format", "black", "prettier", "powershell", "taplo"],
    )
    def test_missing_tool_returns_127_in_every_mode(
        self, monkeypatch, capsys, mode_args, formatter
    ):
        # Missing-executable handling is identical across default,
        # --check, and --dry-run: no traceback, stderr via
        # "formattington: ...", exit 127.
        set_argv(monkeypatch, ["formattington", *mode_args, formatter])
        patch_target = {
            "shfmt": "formattington.shfmt.find_tool",
            "clang-format": "formattington.clang_format.find_tool",
            "black": "formattington.format_python.find_tool",
            "prettier": "formattington.prettier.find_tool",
            "powershell": "formattington.powershell.find_tool",
            "taplo": "formattington.taplo.find_tool",
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
        if formatter == "powershell":
            assert "Could not find 'pwsh' or 'powershell'" in err
        else:
            assert f"Could not find '{formatter}'" in err
        assert "Traceback" not in err

    def test_missing_tool_with_forwarded_args_returns_127(self, monkeypatch, capsys):
        # "--check" after the formatter is forwarded to the formatter,
        # but a missing executable still exits 127 before any run.
        set_argv(monkeypatch, ["formattington", "shfmt", "--check"])
        with patch(
            "formattington.shfmt.find_tool",
            side_effect=FileNotFoundError("Could not find 'shfmt'"),
        ):
            assert main() == 127
        assert "Could not find 'shfmt'" in capsys.readouterr().err

    def test_prettier_missing_with_forwarded_args_returns_127(
        self, monkeypatch, capsys
    ):
        set_argv(monkeypatch, ["formattington", "prettier", "--check"])
        with patch(
            "formattington.prettier.find_tool",
            side_effect=FileNotFoundError("Could not find 'prettier'"),
        ):
            assert main() == 127
        err = capsys.readouterr().err
        assert err.startswith("formattington:")
        assert "Could not find 'prettier'" in err
        assert "Traceback" not in err

    def test_help_and_version_do_not_require_tools(self, monkeypatch):
        # --help and --version must work even when tools are missing.
        set_argv(monkeypatch, ["formattington", "--help"])
        with pytest.raises(SystemExit) as exc_info:
            main()
        assert exc_info.value.code == 0
        set_argv(monkeypatch, ["formattington", "--version"])
        assert main() == 0

    def test_all_returns_127_when_all_missing(self, monkeypatch, capsys):
        set_argv(monkeypatch, ["formattington", "--check", "all"])
        with patch(
            "formattington.shfmt.find_tool",
            side_effect=FileNotFoundError("Could not find 'shfmt'"),
        ), patch(
            "formattington.clang_format.find_tool",
            side_effect=FileNotFoundError("Could not find 'clang-format'"),
        ), patch(
            "formattington.format_python.find_tool",
            side_effect=FileNotFoundError("Could not find 'black'"),
        ), patch(
            "formattington.prettier.find_tool",
            side_effect=FileNotFoundError("Could not find 'prettier'"),
        ), patch(
            "formattington.powershell.find_tool",
            side_effect=FileNotFoundError("Could not find 'pwsh' or 'powershell'"),
        ), patch(
            "formattington.taplo.find_tool",
            side_effect=FileNotFoundError("Could not find 'taplo'"),
        ):
            assert main() == 127
        err = capsys.readouterr().err
        assert err.count("formattington:") == 6


class TestSeriesAndParallel:
    """Issue #3: --series vs concurrent default for 'all'."""

    def test_all_series_after_all_runs_every_formatter_in_order(
        self, monkeypatch, mock_formatters
    ):
        calls: list[str] = []

        def make_side_effect(name):
            def _fake(*args, **kwargs):
                calls.append(name)
                return 0

            return _fake

        for name, mock in mock_formatters.items():
            mock.side_effect = make_side_effect(name)
        set_argv(monkeypatch, ["formattington", "all", "--series"])
        assert main() == 0
        assert calls == ["shfmt", "clang-format", "black", "prettier", "powershell", "taplo"]
        for mock in mock_formatters.values():
            mock.assert_called_once_with(check=False, dry_run=False, args=[], verbose=False)

    def test_series_before_all_runs_every_formatter(self, monkeypatch, mock_formatters):
        set_argv(monkeypatch, ["formattington", "--series", "all"])
        assert main() == 0
        for mock in mock_formatters.values():
            mock.assert_called_once_with(check=False, dry_run=False, args=[], verbose=False)

    def test_series_flag_not_forwarded_to_formatters(self, monkeypatch, mock_formatters):
        set_argv(monkeypatch, ["formattington", "all", "--series"])
        assert main() == 0
        for mock in mock_formatters.values():
            assert mock.call_args.kwargs["args"] == []

    def test_series_with_check_before_all(self, monkeypatch, mock_formatters):
        set_argv(monkeypatch, ["formattington", "--check", "--series", "all"])
        assert main() == 0
        for mock in mock_formatters.values():
            mock.assert_called_once_with(check=True, dry_run=False, args=[], verbose=False)

    def test_check_before_all_series_after(self, monkeypatch, mock_formatters):
        set_argv(monkeypatch, ["formattington", "--check", "all", "--series"])
        assert main() == 0
        for mock in mock_formatters.values():
            mock.assert_called_once_with(check=True, dry_run=False, args=[], verbose=False)

    def test_all_series_with_extra_arg_exits_2(self, monkeypatch, mock_formatters):
        set_argv(monkeypatch, ["formattington", "all", "--series", "extra"])
        with pytest.raises(SystemExit) as exc_info:
            main()
        assert exc_info.value.code == 2

    def test_series_is_noop_for_single_formatter(self, monkeypatch, mock_formatters):
        set_argv(monkeypatch, ["formattington", "--series", "shfmt"])
        assert main() == 0
        mock_formatters["shfmt"].assert_called_once_with(
            check=False, dry_run=False, args=[], verbose=False
        )

    def test_series_after_single_formatter_is_forwarded(
        self, monkeypatch, mock_formatters
    ):
        # Like --check after a formatter, "--series" after a single formatter
        # belongs to that formatter, not to Formattington.
        set_argv(monkeypatch, ["formattington", "shfmt", "--series"])
        assert main() == 0
        mock_formatters["shfmt"].assert_called_once_with(
            check=False, dry_run=False, args=["--series"], verbose=False
        )

    def test_series_without_formatter_is_a_noop(self, monkeypatch, mock_formatters):
        set_argv(monkeypatch, ["formattington", "--series"])
        assert main() == 0
        for mock in mock_formatters.values():
            mock.assert_not_called()

    def test_help_lists_series_and_documents_exit_rule(self, monkeypatch, capsys):
        set_argv(monkeypatch, ["formattington", "--help"])
        with pytest.raises(SystemExit) as exc_info:
            main()
        assert exc_info.value.code == 0
        out = capsys.readouterr().out
        assert "--series" in out
        assert "concurrently" in out
        assert "last non-zero" in out

    def test_default_all_runs_concurrently(self, monkeypatch, mock_formatters):
        # A barrier that only releases when all formatters overlap proves
        # the default is concurrent (series would deadlock/time out).
        import threading

        barrier = threading.Barrier(len(mock_formatters), timeout=10)

        def make_side_effect():
            def _fake(*args, **kwargs):
                barrier.wait()
                return 0

            return _fake

        for mock in mock_formatters.values():
            mock.side_effect = make_side_effect()
        set_argv(monkeypatch, ["formattington", "all"])
        assert main() == 0

    def test_series_runs_sequentially_same_thread(self, monkeypatch, mock_formatters):
        import threading

        idents: list[int] = []

        def make_side_effect():
            def _fake(*args, **kwargs):
                idents.append(threading.get_ident())
                return 0

            return _fake

        for mock in mock_formatters.values():
            mock.side_effect = make_side_effect()
        set_argv(monkeypatch, ["formattington", "all", "--series"])
        main_thread = threading.get_ident()
        assert main() == 0
        assert len(set(idents)) == 1
        assert idents[0] == main_thread

    def test_default_all_aggregates_last_nonzero(self, monkeypatch, mock_formatters):
        mock_formatters["shfmt"].return_value = 5
        mock_formatters["clang-format"].return_value = 2
        mock_formatters["black"].return_value = 0
        mock_formatters["prettier"].return_value = 1
        set_argv(monkeypatch, ["formattington", "all"])
        assert main() == 1

    def test_series_aggregates_last_nonzero(self, monkeypatch, mock_formatters):
        mock_formatters["shfmt"].return_value = 5
        mock_formatters["prettier"].return_value = 1
        set_argv(monkeypatch, ["formattington", "all", "--series"])
        assert main() == 1

    def test_failing_formatter_does_not_swallow_others(self, monkeypatch, mock_formatters, capsys):
        mock_formatters["shfmt"].return_value = 3
        set_argv(monkeypatch, ["formattington", "all"])
        assert main() == 3
        for mock in mock_formatters.values():
            mock.assert_called_once_with(check=False, dry_run=False, args=[], verbose=False)

    def test_series_failing_does_not_swallow_others(self, monkeypatch, mock_formatters):
        mock_formatters["prettier"].return_value = 4
        set_argv(monkeypatch, ["formattington", "all", "--series"])
        assert main() == 4
        for mock in mock_formatters.values():
            mock.assert_called_once_with(check=False, dry_run=False, args=[], verbose=False)

    def test_parallel_missing_does_not_swallow_others(self, monkeypatch, mock_formatters, capsys):
        mock_formatters["shfmt"].side_effect = FileNotFoundError(
            "Could not find 'shfmt'"
        )
        set_argv(monkeypatch, ["formattington", "all"])
        assert main() == 127
        for name in ("clang-format", "black", "prettier"):
            mock_formatters[name].assert_called_once()
        assert "Could not find 'shfmt'" in capsys.readouterr().err

    def test_series_missing_does_not_swallow_others(self, monkeypatch, mock_formatters, capsys):
        mock_formatters["black"].side_effect = FileNotFoundError(
            "Could not find 'black'"
        )
        set_argv(monkeypatch, ["formattington", "all", "--series"])
        assert main() == 127
        for name in ("shfmt", "clang-format", "prettier"):
            mock_formatters[name].assert_called_once()
        assert "Could not find 'black'" in capsys.readouterr().err

    def test_missing_in_middle_last_nonzero_wins(self, monkeypatch, mock_formatters):
        # Missing (127) counts in its order slot: a later formatter
        # failure (black=2) wins over the earlier missing code.
        mock_formatters["clang-format"].side_effect = FileNotFoundError(
            "Could not find 'clang-format'"
        )
        mock_formatters["black"].return_value = 2
        set_argv(monkeypatch, ["formattington", "all"])
        assert main() == 2

    def test_check_mode_all_still_reports_every_formatter(
        self, monkeypatch, mock_formatters
    ):
        # Concurrent default and --series both run every formatter in
        # check mode.
        mock_formatters["black"].return_value = 1
        set_argv(monkeypatch, ["formattington", "--check", "all"])
        assert main() == 1
        for mock in mock_formatters.values():
            mock.assert_called_once_with(check=True, dry_run=False, args=[], verbose=False)
