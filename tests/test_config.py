"""Tests for the .formattington.json configuration file (issue #31).

When a config is in effect, ``formattington all`` runs exactly the JSON
``commands`` list instead of the full formatter registry. Tests continue
the existing style: formatter mocks plus tempdirs, no network and no
real formatter binaries (except one real-discovery test that exercises
the actual ``discover_files`` helper against a temporary directory).
"""

from __future__ import annotations

import json
import sys
import threading
from pathlib import Path
from unittest.mock import MagicMock

import pytest

import formattington.cli as cli
from formattington.cli import main


@pytest.fixture
def mock_formatters(monkeypatch):
    """Replace FORMATTERS with mocks; stub discovery to return no files."""
    mocks = {
        name: MagicMock(return_value=0)
        for name in ("shfmt", "clang-format", "black", "prettier", "powershell", "taplo")
    }
    monkeypatch.setattr(cli, "discover_files", lambda extensions, root=None: [])
    original = dict(cli.FORMATTERS)
    cli.FORMATTERS.update(mocks)
    try:
        yield mocks
    finally:
        cli.FORMATTERS.clear()
        cli.FORMATTERS.update(original)


def _set_argv(monkeypatch, argv):
    monkeypatch.setattr(sys, "argv", argv)


def _write(path: Path, text: str) -> Path:
    path.write_text(text, encoding="utf-8")
    return path


def _write_config(path: Path, payload) -> Path:
    return _write(path, json.dumps(payload))


def _isolate(monkeypatch, tmp_path: Path):
    monkeypatch.chdir(tmp_path)
    monkeypatch.delenv("FORMATTINGTON_CONFIG_PATH", raising=False)


class TestDiscoveryPrecedence:
    def test_config_flag_after_all(self, monkeypatch, tmp_path, mock_formatters):
        _isolate(monkeypatch, tmp_path)
        cfg = _write_config(tmp_path / "cfg.json", {"commands": [["black", "--check"]]})
        _set_argv(monkeypatch, ["formattington", "all", "--config", str(cfg)])
        assert main() == 0
        mock_formatters["black"].assert_called_once_with(
            check=False, dry_run=False, args=["--check"]
        )
        mock_formatters["shfmt"].assert_not_called()

    def test_config_flag_before_all(self, monkeypatch, tmp_path, mock_formatters):
        _isolate(monkeypatch, tmp_path)
        cfg = _write_config(tmp_path / "cfg.json", {"commands": [["shfmt"]]})
        _set_argv(monkeypatch, ["formattington", "--config", str(cfg), "all"])
        assert main() == 0
        mock_formatters["shfmt"].assert_called_once_with(
            check=False, dry_run=False, args=[]
        )
        mock_formatters["black"].assert_not_called()

    def test_config_equals_form_after_all(self, monkeypatch, tmp_path, mock_formatters):
        _isolate(monkeypatch, tmp_path)
        cfg = _write_config(tmp_path / "cfg.json", {"commands": [["black"]]})
        _set_argv(monkeypatch, ["formattington", "all", f"--config={cfg}"])
        assert main() == 0
        mock_formatters["black"].assert_called_once()

    def test_env_var_discovers_config(self, monkeypatch, tmp_path, mock_formatters):
        _isolate(monkeypatch, tmp_path)
        cfg = _write_config(tmp_path / "env.json", {"commands": [["black"]]})
        monkeypatch.setenv("FORMATTINGTON_CONFIG_PATH", str(cfg))
        _set_argv(monkeypatch, ["formattington", "all"])
        assert main() == 0
        mock_formatters["black"].assert_called_once()

    def test_auto_discovered_config(self, monkeypatch, tmp_path, mock_formatters):
        _isolate(monkeypatch, tmp_path)
        _write_config(tmp_path / ".formattington.json", {"commands": [["shfmt", "-i", "2"]]})
        _set_argv(monkeypatch, ["formattington", "all"])
        assert main() == 0
        mock_formatters["shfmt"].assert_called_once_with(
            check=False, dry_run=False, args=["-i", "2"]
        )
        mock_formatters["black"].assert_not_called()

    def test_no_config_falls_back_to_full_registry(self, monkeypatch, tmp_path, mock_formatters):
        _isolate(monkeypatch, tmp_path)
        _set_argv(monkeypatch, ["formattington", "all"])
        assert main() == 0
        for mock in mock_formatters.values():
            mock.assert_called_once_with(check=False, dry_run=False, args=[])

    def test_flag_beats_env_and_auto(self, monkeypatch, tmp_path, mock_formatters):
        _isolate(monkeypatch, tmp_path)
        _write_config(tmp_path / ".formattington.json", {"commands": [["shfmt"]]})
        env_cfg = _write_config(tmp_path / "env.json", {"commands": [["clang-format"]]})
        flag_cfg = _write_config(tmp_path / "flag.json", {"commands": [["black"]]})
        monkeypatch.setenv("FORMATTINGTON_CONFIG_PATH", str(env_cfg))
        _set_argv(monkeypatch, ["formattington", "all", "--config", str(flag_cfg)])
        assert main() == 0
        mock_formatters["black"].assert_called_once()
        mock_formatters["shfmt"].assert_not_called()
        mock_formatters["clang-format"].assert_not_called()

    def test_flag_wins_even_when_env_missing(self, monkeypatch, tmp_path, mock_formatters):
        _isolate(monkeypatch, tmp_path)
        flag_cfg = _write_config(tmp_path / "flag.json", {"commands": [["black"]]})
        monkeypatch.setenv("FORMATTINGTON_CONFIG_PATH", str(tmp_path / "missing.json"))
        _set_argv(monkeypatch, ["formattington", "--config", str(flag_cfg), "all"])
        assert main() == 0
        mock_formatters["black"].assert_called_once()

    def test_env_beats_auto(self, monkeypatch, tmp_path, mock_formatters):
        _isolate(monkeypatch, tmp_path)
        _write_config(tmp_path / ".formattington.json", {"commands": [["shfmt"]]})
        env_cfg = _write_config(tmp_path / "env.json", {"commands": [["black"]]})
        monkeypatch.setenv("FORMATTINGTON_CONFIG_PATH", str(env_cfg))
        _set_argv(monkeypatch, ["formattington", "all"])
        assert main() == 0
        mock_formatters["black"].assert_called_once()
        mock_formatters["shfmt"].assert_not_called()

    def test_empty_env_falls_through_to_auto(self, monkeypatch, tmp_path, mock_formatters):
        _isolate(monkeypatch, tmp_path)
        _write_config(tmp_path / ".formattington.json", {"commands": [["shfmt"]]})
        monkeypatch.setenv("FORMATTINGTON_CONFIG_PATH", "")
        _set_argv(monkeypatch, ["formattington", "all"])
        assert main() == 0
        mock_formatters["shfmt"].assert_called_once()
        mock_formatters["black"].assert_not_called()

    def test_empty_env_no_auto_falls_back(self, monkeypatch, tmp_path, mock_formatters):
        _isolate(monkeypatch, tmp_path)
        monkeypatch.setenv("FORMATTINGTON_CONFIG_PATH", "")
        _set_argv(monkeypatch, ["formattington", "all"])
        assert main() == 0
        for mock in mock_formatters.values():
            mock.assert_called_once()

    def test_relative_config_path_against_cwd(self, monkeypatch, tmp_path, mock_formatters):
        _isolate(monkeypatch, tmp_path)
        _write_config(tmp_path / "cfg.json", {"commands": [["black"]]})
        _set_argv(monkeypatch, ["formattington", "all", "--config", "cfg.json"])
        assert main() == 0
        mock_formatters["black"].assert_called_once()

    def test_config_path_with_spaces(self, monkeypatch, tmp_path, mock_formatters):
        _isolate(monkeypatch, tmp_path)
        sub = tmp_path / "dir with spaces"
        sub.mkdir()
        cfg = _write_config(sub / "my config.json", {"commands": [["black"]]})
        _set_argv(monkeypatch, ["formattington", "all", "--config", str(cfg)])
        assert main() == 0
        mock_formatters["black"].assert_called_once()

    def test_flag_missing_file_exits_2(self, monkeypatch, tmp_path, mock_formatters, capsys):
        _isolate(monkeypatch, tmp_path)
        missing = "missing-config.json"
        _set_argv(monkeypatch, ["formattington", "all", "--config", missing])
        assert main() == 2
        err = capsys.readouterr().err
        assert err.startswith("formattington:")
        assert missing in err
        assert "--config" in err
        assert "Traceback" not in err
        for mock in mock_formatters.values():
            mock.assert_not_called()

    def test_flag_not_a_file_exits_2(self, monkeypatch, tmp_path, mock_formatters, capsys):
        _isolate(monkeypatch, tmp_path)
        sub = tmp_path / "adir"
        sub.mkdir()
        _set_argv(monkeypatch, ["formattington", "all", "--config", str(sub)])
        assert main() == 2
        err = capsys.readouterr().err
        assert err.startswith("formattington:")
        assert str(sub) in err
        assert "--config" in err

    def test_env_missing_file_exits_2(self, monkeypatch, tmp_path, mock_formatters, capsys):
        _isolate(monkeypatch, tmp_path)
        missing = str(tmp_path / "missing.json")
        monkeypatch.setenv("FORMATTINGTON_CONFIG_PATH", missing)
        _set_argv(monkeypatch, ["formattington", "all"])
        assert main() == 2
        err = capsys.readouterr().err
        assert err.startswith("formattington:")
        assert missing in err
        assert "FORMATTINGTON_CONFIG_PATH" in err
        assert "Traceback" not in err

    def test_malformed_auto_exits_2_without_fallback(
        self, monkeypatch, tmp_path, mock_formatters, capsys
    ):
        _isolate(monkeypatch, tmp_path)
        _write(tmp_path / ".formattington.json", "not json at all")
        _set_argv(monkeypatch, ["formattington", "all"])
        assert main() == 2
        err = capsys.readouterr().err
        assert err.startswith("formattington:")
        assert ".formattington.json" in err
        assert "Traceback" not in err
        for mock in mock_formatters.values():
            mock.assert_not_called()

    def test_config_value_equal_to_formatter_name(self, monkeypatch, tmp_path, mock_formatters):
        # A config path literally named "black" must not be mistaken for
        # the formatter selector.
        _isolate(monkeypatch, tmp_path)
        _write_config(tmp_path / "black", {"commands": [["shfmt"]]})
        _set_argv(monkeypatch, ["formattington", "--config", "black", "all"])
        assert main() == 0
        mock_formatters["shfmt"].assert_called_once()
        mock_formatters["black"].assert_not_called()


class TestSchemaValidation:
    @pytest.mark.parametrize(
        "payload",
        [
            pytest.param("not json at all", id="bad-json"),
            pytest.param(json.dumps([["black"]]), id="bare-array"),
            pytest.param(json.dumps("black"), id="top-string"),
            pytest.param(json.dumps(42), id="top-number"),
            pytest.param(json.dumps({}), id="missing-commands"),
            pytest.param(json.dumps({"commands": "black"}), id="non-list-commands"),
            pytest.param(json.dumps({"commands": ["black --check"]}), id="shell-string"),
            pytest.param(json.dumps({"commands": [{"tool": "black"}]}), id="object-entry"),
            pytest.param(json.dumps({"commands": [42]}), id="non-list-entry"),
            pytest.param(json.dumps({"commands": [[]]}), id="empty-argv"),
            pytest.param(json.dumps({"commands": [["black", 42]]}), id="non-string-element"),
            pytest.param(json.dumps({"commands": [["nosuch"]]}), id="unknown-tool"),
            pytest.param(json.dumps({"commands": [["all"]]}), id="all-not-allowed"),
            pytest.param(
                json.dumps({"commands": [], "extr": 1}), id="unknown-top-key"
            ),
        ],
    )
    def test_invalid_schemas_exit_2(self, monkeypatch, tmp_path, mock_formatters, capsys, payload):
        _isolate(monkeypatch, tmp_path)
        cfg = _write(tmp_path / "cfg.json", payload)
        _set_argv(monkeypatch, ["formattington", "all", "--config", str(cfg)])
        assert main() == 2
        err = capsys.readouterr().err
        assert err.startswith("formattington:")
        assert "cfg.json" in err
        assert "Traceback" not in err
        for mock in mock_formatters.values():
            mock.assert_not_called()

    def test_unknown_tool_names_entry_index(self, monkeypatch, tmp_path, mock_formatters, capsys):
        _isolate(monkeypatch, tmp_path)
        cfg = _write_config(tmp_path / "cfg.json", {"commands": [["black"], ["nosuch", "--x"]]})
        _set_argv(monkeypatch, ["formattington", "all", "--config", str(cfg)])
        assert main() == 2
        err = capsys.readouterr().err
        assert "commands[1]" in err
        assert "nosuch" in err
        assert "cfg.json" in err

    def test_alias_with_missing_backend_is_not_yet_supported(
        self, monkeypatch, tmp_path, mock_formatters, capsys
    ):
        _isolate(monkeypatch, tmp_path)
        monkeypatch.setitem(cli.LANGUAGE_ALIASES, "cobol", "cobol-formatter")
        cfg = _write_config(tmp_path / "cfg.json", {"commands": [["cobol"]]})
        _set_argv(monkeypatch, ["formattington", "all", "--config", str(cfg)])
        assert main() == 2
        err = capsys.readouterr().err
        assert "not yet supported" in err
        assert "commands[0]" in err

    def test_empty_commands_valid_exits_0_no_runs(self, monkeypatch, tmp_path, mock_formatters):
        _isolate(monkeypatch, tmp_path)
        cfg = _write_config(tmp_path / "cfg.json", {"commands": []})
        _set_argv(monkeypatch, ["formattington", "all", "--config", str(cfg)])
        assert main() == 0
        for mock in mock_formatters.values():
            mock.assert_not_called()

    def test_empty_commands_valid_with_series_exits_0_no_runs(
        self, monkeypatch, tmp_path, mock_formatters
    ):
        _isolate(monkeypatch, tmp_path)
        cfg = _write_config(tmp_path / "cfg.json", {"commands": []})
        _set_argv(monkeypatch, ["formattington", "all", "--series", "--config", str(cfg)])
        assert main() == 0
        for mock in mock_formatters.values():
            mock.assert_not_called()


class TestExecution:
    def test_runs_exactly_listed_commands(self, monkeypatch, tmp_path, mock_formatters):
        _isolate(monkeypatch, tmp_path)
        cfg = _write_config(
            tmp_path / "cfg.json",
            {"commands": [["black", "--check", "src"], ["prettier", "docs/**/*.md"]]},
        )
        _set_argv(monkeypatch, ["formattington", "all", "--config", str(cfg)])
        assert main() == 0
        mock_formatters["black"].assert_called_once_with(
            check=False, dry_run=False, args=["--check", "src"]
        )
        mock_formatters["prettier"].assert_called_once_with(
            check=False, dry_run=False, args=["docs/**/*.md"]
        )
        mock_formatters["shfmt"].assert_not_called()
        mock_formatters["clang-format"].assert_not_called()

    def test_alias_commands_resolve(self, monkeypatch, tmp_path, mock_formatters):
        _isolate(monkeypatch, tmp_path)
        cfg = _write_config(
            tmp_path / "cfg.json",
            {"commands": [["bash"], ["python"], ["md"], ["c"], ["ps1"], ["toml"]]},
        )
        _set_argv(monkeypatch, ["formattington", "all", "--config", str(cfg)])
        assert main() == 0
        mock_formatters["shfmt"].assert_called_once()
        mock_formatters["black"].assert_called_once()
        mock_formatters["prettier"].assert_called_once()
        mock_formatters["clang-format"].assert_called_once()
        mock_formatters["powershell"].assert_called_once()
        mock_formatters["taplo"].assert_called_once()

    def test_forwards_spaces_as_single_entries(self, monkeypatch, tmp_path, mock_formatters):
        _isolate(monkeypatch, tmp_path)
        cfg = _write_config(
            tmp_path / "cfg.json",
            {"commands": [["prettier", "my file.md"], ["shfmt", "-i", "2", "sub dir/my file.sh"]]},
        )
        _set_argv(monkeypatch, ["formattington", "all", "--config", str(cfg)])
        assert main() == 0
        mock_formatters["prettier"].assert_called_once_with(
            check=False, dry_run=False, args=["my file.md"]
        )
        mock_formatters["shfmt"].assert_called_once_with(
            check=False, dry_run=False, args=["-i", "2", "sub dir/my file.sh"]
        )

    def test_check_propagates_to_config_commands(self, monkeypatch, tmp_path, mock_formatters):
        _isolate(monkeypatch, tmp_path)
        cfg = _write_config(tmp_path / "cfg.json", {"commands": [["black"], ["shfmt"]]})
        _set_argv(monkeypatch, ["formattington", "--check", "all", "--config", str(cfg)])
        assert main() == 0
        for name in ("black", "shfmt"):
            mock_formatters[name].assert_called_once_with(
                check=True, dry_run=False, args=[]
            )

    def test_dry_run_and_verbose_propagate(self, monkeypatch, tmp_path, mock_formatters):
        _isolate(monkeypatch, tmp_path)
        cfg = _write_config(tmp_path / "cfg.json", {"commands": [["black"], ["prettier"]]})
        _set_argv(
            monkeypatch,
            ["formattington", "--dry-run", "--verbose", "all", "--config", str(cfg)],
        )
        assert main() == 0
        for name in ("black", "prettier"):
            mock_formatters[name].assert_called_once_with(
                check=False, dry_run=True, args=[], verbose=True
            )

    def test_series_preserves_file_order_same_thread(self, monkeypatch, tmp_path, mock_formatters):
        _isolate(monkeypatch, tmp_path)
        cfg = _write_config(
            tmp_path / "cfg.json",
            {"commands": [["black"], ["shfmt"], ["prettier"]]},
        )
        calls: list[str] = []
        idents: list[int] = []

        def make_side_effect(name):
            def _fake(*args, **kwargs):
                calls.append(name)
                idents.append(threading.get_ident())
                return 0

            return _fake

        for name in ("black", "shfmt", "prettier"):
            mock_formatters[name].side_effect = make_side_effect(name)
        _set_argv(monkeypatch, ["formattington", "all", "--series", "--config", str(cfg)])
        assert main() == 0
        assert calls == ["black", "shfmt", "prettier"]
        assert len(set(idents)) == 1
        assert idents[0] == threading.get_ident()

    def test_default_runs_concurrently(self, monkeypatch, tmp_path, mock_formatters):
        _isolate(monkeypatch, tmp_path)
        cfg = _write_config(
            tmp_path / "cfg.json", {"commands": [["black"], ["shfmt"], ["prettier"]]}
        )
        barrier = threading.Barrier(3, timeout=10)

        def _fake(*args, **kwargs):
            barrier.wait()
            return 0

        for name in ("black", "shfmt", "prettier"):
            mock_formatters[name].side_effect = _fake
        _set_argv(monkeypatch, ["formattington", "all", "--config", str(cfg)])
        assert main() == 0
        mock_formatters["black"].assert_called_once()
        mock_formatters["shfmt"].assert_called_once()
        mock_formatters["prettier"].assert_called_once()

    def test_duplicate_tools_run_per_command(self, monkeypatch, tmp_path, mock_formatters):
        _isolate(monkeypatch, tmp_path)
        cfg = _write_config(
            tmp_path / "cfg.json",
            {"commands": [["black", "--check"], ["black", "--diff"]]},
        )
        _set_argv(monkeypatch, ["formattington", "all", "--series", "--config", str(cfg)])
        assert main() == 0
        assert mock_formatters["black"].call_count == 2
        seen = [call.kwargs["args"] for call in mock_formatters["black"].call_args_list]
        assert seen == [["--check"], ["--diff"]]

    def test_failing_does_not_swallow_later_commands(self, monkeypatch, tmp_path, mock_formatters):
        _isolate(monkeypatch, tmp_path)
        mock_formatters["black"].return_value = 3
        cfg = _write_config(tmp_path / "cfg.json", {"commands": [["black"], ["shfmt"]]})
        _set_argv(monkeypatch, ["formattington", "all", "--config", str(cfg)])
        assert main() == 3
        mock_formatters["black"].assert_called_once()
        mock_formatters["shfmt"].assert_called_once()

    def test_missing_does_not_swallow_later_commands(
        self, monkeypatch, tmp_path, mock_formatters, capsys
    ):
        _isolate(monkeypatch, tmp_path)
        mock_formatters["black"].side_effect = FileNotFoundError("Could not find 'black'")
        cfg = _write_config(tmp_path / "cfg.json", {"commands": [["black"], ["shfmt"]]})
        _set_argv(monkeypatch, ["formattington", "all", "--config", str(cfg)])
        assert main() == 127
        mock_formatters["shfmt"].assert_called_once()
        err = capsys.readouterr().err
        assert err.startswith("formattington:")
        assert "Could not find 'black'" in err
        assert "Traceback" not in err

    def test_missing_does_not_swallow_in_series(
        self, monkeypatch, tmp_path, mock_formatters, capsys
    ):
        _isolate(monkeypatch, tmp_path)
        mock_formatters["black"].side_effect = FileNotFoundError("Could not find 'black'")
        cfg = _write_config(tmp_path / "cfg.json", {"commands": [["shfmt"], ["black"], ["prettier"]]})
        _set_argv(monkeypatch, ["formattington", "all", "--series", "--config", str(cfg)])
        assert main() == 127
        mock_formatters["shfmt"].assert_called_once()
        mock_formatters["prettier"].assert_called_once()

    @pytest.mark.parametrize(
        ("codes", "expected"),
        [([5, 1], 1), ([1, 5], 5)],
        ids=["last-wins-1", "last-wins-5"],
    )
    def test_exit_aggregation_follows_config_order(
        self, monkeypatch, tmp_path, mock_formatters, codes, expected
    ):
        _isolate(monkeypatch, tmp_path)
        mock_formatters["black"].return_value = codes[0]
        mock_formatters["shfmt"].return_value = codes[1]
        cfg = _write_config(tmp_path / "cfg.json", {"commands": [["black"], ["shfmt"]]})
        _set_argv(monkeypatch, ["formattington", "all", "--series", "--config", str(cfg)])
        assert main() == expected

    def test_exit_aggregation_parallel_follows_config_order(self, monkeypatch, tmp_path, mock_formatters):
        _isolate(monkeypatch, tmp_path)
        mock_formatters["black"].return_value = 5
        mock_formatters["prettier"].return_value = 1
        cfg = _write_config(tmp_path / "cfg.json", {"commands": [["black"], ["prettier"]]})
        _set_argv(monkeypatch, ["formattington", "all", "--config", str(cfg)])
        assert main() == 1

    def test_config_discovery_appends_files(self, monkeypatch, tmp_path):
        # Real discover_files (not stubbed): a bare ["shfmt"] command
        # discovers shell files like a single-tool run does.
        _isolate(monkeypatch, tmp_path)
        (tmp_path / "a.sh").write_text("#!/bin/sh\n")
        (tmp_path / "b.sh").write_text("#!/bin/sh\n")
        (tmp_path / "c.txt").write_text("hi\n")
        cfg = _write_config(tmp_path / "cfg.json", {"commands": [["shfmt"]]})
        mocks = {
            name: MagicMock(return_value=0)
            for name in ("shfmt", "clang-format", "black", "prettier", "powershell", "taplo")
        }
        original = dict(cli.FORMATTERS)
        cli.FORMATTERS.update(mocks)
        try:
            _set_argv(monkeypatch, ["formattington", "all", "--config", str(cfg)])
            assert main() == 0
        finally:
            cli.FORMATTERS.clear()
            cli.FORMATTERS.update(original)
        assert mocks["shfmt"].call_args.kwargs["args"] == ["a.sh", "b.sh"]

    def test_config_explicit_files_suppress_discovery(self, monkeypatch, tmp_path):
        _isolate(monkeypatch, tmp_path)
        (tmp_path / "a.sh").write_text("#!/bin/sh\n")
        (tmp_path / "explicit.sh").write_text("#!/bin/sh\n")
        cfg = _write_config(tmp_path / "cfg.json", {"commands": [["shfmt", "explicit.sh"]]})
        mocks = {
            name: MagicMock(return_value=0)
            for name in ("shfmt", "clang-format", "black", "prettier", "powershell", "taplo")
        }
        original = dict(cli.FORMATTERS)
        cli.FORMATTERS.update(mocks)
        try:
            _set_argv(monkeypatch, ["formattington", "all", "--config", str(cfg)])
            assert main() == 0
        finally:
            cli.FORMATTERS.clear()
            cli.FORMATTERS.update(original)
        assert mocks["shfmt"].call_args.kwargs["args"] == ["explicit.sh"]


class TestSingleToolInteraction:
    def test_flag_before_formatter_exits_2(self, monkeypatch, tmp_path, mock_formatters, capsys):
        _isolate(monkeypatch, tmp_path)
        cfg = _write_config(tmp_path / "cfg.json", {"commands": [["black"]]})
        _set_argv(monkeypatch, ["formattington", "--config", str(cfg), "black"])
        with pytest.raises(SystemExit) as exc_info:
            main()
        assert exc_info.value.code == 2
        assert "--config" in capsys.readouterr().err
        for mock in mock_formatters.values():
            mock.assert_not_called()

    def test_flag_before_alias_exits_2(self, monkeypatch, tmp_path, mock_formatters):
        _isolate(monkeypatch, tmp_path)
        cfg = _write_config(tmp_path / "cfg.json", {"commands": [["black"]]})
        _set_argv(monkeypatch, ["formattington", "--config", str(cfg), "python"])
        with pytest.raises(SystemExit) as exc_info:
            main()
        assert exc_info.value.code == 2

    def test_config_after_single_is_forwarded(self, monkeypatch, tmp_path, mock_formatters):
        # Arguments after a single formatter belong to that formatter,
        # so Prettier's native --config keeps working.
        _isolate(monkeypatch, tmp_path)
        _set_argv(monkeypatch, ["formattington", "prettier", "--config", "my.json"])
        assert main() == 0
        mock_formatters["prettier"].assert_called_once_with(
            check=False, dry_run=False, args=["--config", "my.json"]
        )

    def test_single_ignores_auto_config(self, monkeypatch, tmp_path, mock_formatters):
        _isolate(monkeypatch, tmp_path)
        _write_config(tmp_path / ".formattington.json", {"commands": [["black"]]})
        _set_argv(monkeypatch, ["formattington", "shfmt"])
        assert main() == 0
        mock_formatters["shfmt"].assert_called_once()
        mock_formatters["black"].assert_not_called()

    def test_single_ignores_missing_env_var(self, monkeypatch, tmp_path, mock_formatters):
        _isolate(monkeypatch, tmp_path)
        monkeypatch.setenv("FORMATTINGTON_CONFIG_PATH", str(tmp_path / "missing.json"))
        _set_argv(monkeypatch, ["formattington", "shfmt"])
        assert main() == 0
        mock_formatters["shfmt"].assert_called_once()

    def test_single_ignores_valid_env_config(self, monkeypatch, tmp_path, mock_formatters):
        _isolate(monkeypatch, tmp_path)
        cfg = _write_config(tmp_path / "env.json", {"commands": [["black"]]})
        monkeypatch.setenv("FORMATTINGTON_CONFIG_PATH", str(cfg))
        _set_argv(monkeypatch, ["formattington", "shfmt"])
        assert main() == 0
        mock_formatters["shfmt"].assert_called_once()
        mock_formatters["black"].assert_not_called()

    def test_extra_args_with_config_exits_2(self, monkeypatch, tmp_path, mock_formatters):
        _isolate(monkeypatch, tmp_path)
        cfg = _write_config(tmp_path / "cfg.json", {"commands": [["black"]]})
        _set_argv(monkeypatch, ["formattington", "all", "--config", str(cfg), "extra"])
        with pytest.raises(SystemExit) as exc_info:
            main()
        assert exc_info.value.code == 2

    def test_duplicate_config_exits_2(self, monkeypatch, tmp_path, mock_formatters):
        _isolate(monkeypatch, tmp_path)
        cfg = _write_config(tmp_path / "cfg.json", {"commands": [["black"]]})
        _set_argv(
            monkeypatch,
            ["formattington", "--config", str(cfg), "all", "--config", str(cfg)],
        )
        with pytest.raises(SystemExit) as exc_info:
            main()
        assert exc_info.value.code == 2

    def test_missing_value_after_all_exits_2(self, monkeypatch, tmp_path, mock_formatters):
        _isolate(monkeypatch, tmp_path)
        _set_argv(monkeypatch, ["formattington", "all", "--config"])
        with pytest.raises(SystemExit) as exc_info:
            main()
        assert exc_info.value.code == 2


class TestHelpAndVersion:
    def test_help_documents_config(self, monkeypatch, tmp_path, capsys):
        _isolate(monkeypatch, tmp_path)
        _set_argv(monkeypatch, ["formattington", "--help"])
        with pytest.raises(SystemExit) as exc_info:
            main()
        assert exc_info.value.code == 0
        out = capsys.readouterr().out
        assert "--config" in out
        assert "FORMATTINGTON_CONFIG_PATH" in out
        assert ".formattington.json" in out

    def test_version_works_without_config(self, monkeypatch, tmp_path, capsys):
        _isolate(monkeypatch, tmp_path)
        _set_argv(monkeypatch, ["formattington", "--version"])
        assert main() == 0
        assert "0.1.0" in capsys.readouterr().out

    def test_version_with_config_flag_works(self, monkeypatch, tmp_path, capsys):
        _isolate(monkeypatch, tmp_path)
        _set_argv(monkeypatch, ["formattington", "--config", "anything.json", "--version"])
        assert main() == 0
        assert "0.1.0" in capsys.readouterr().out
