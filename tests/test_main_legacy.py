"""Tests for the legacy main.main() CLI entry point."""

from __future__ import annotations

import sys
from unittest.mock import MagicMock, patch

import pytest

import formattington.main as main_mod


def _set_argv(monkeypatch, argv):
    monkeypatch.setattr(sys, "argv", argv)


def test_shfmt_selection(monkeypatch):
    _set_argv(monkeypatch, ["formattington", "shfmt"])
    with patch(
        "formattington.main.run_formatter", return_value=0
    ) as mock_run_formatter:
        assert main_mod.main() == 0
        mock_run_formatter.assert_called_once_with("shfmt", None)


def test_check_mode_maps_to_check(monkeypatch):
    _set_argv(monkeypatch, ["formattington", "--check", "shfmt"])
    with patch("formattington.main.run_formatter", return_value=0) as mock:
        assert main_mod.main() == 0
        mock.assert_called_once_with("shfmt", "check")


def test_dry_run_mode(monkeypatch):
    _set_argv(monkeypatch, ["formattington", "--dry-run", "shfmt"])
    with patch("formattington.main.run_formatter", return_value=0) as mock:
        assert main_mod.main() == 0
        mock.assert_called_once_with("shfmt", "dry-run")


def test_all_runs_all_formatters(monkeypatch):
    _set_argv(monkeypatch, ["formattington", "all"])
    with patch("formattington.main.run_formatter", return_value=0) as mock:
        assert main_mod.main() == 0
        called_names = [call.args[0] for call in mock.call_args_list]
        assert called_names == list(main_mod.FORMATTERS)


def test_unknown_formatter_exits_2(monkeypatch):
    _set_argv(monkeypatch, ["formattington", "nope"])
    with pytest.raises(SystemExit) as exc_info:
        main_mod.main()
    assert exc_info.value.code == 2


def test_check_and_dry_run_exclusive(monkeypatch):
    _set_argv(monkeypatch, ["formattington", "--check", "--dry-run", "shfmt"])
    with pytest.raises(SystemExit) as exc_info:
        main_mod.main()
    assert exc_info.value.code == 2


def test_preserves_nonzero_exit(monkeypatch):
    _set_argv(monkeypatch, ["formattington", "shfmt"])
    with patch("formattington.main.run_formatter", return_value=4):
        assert main_mod.main() == 4
