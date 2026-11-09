"""Tests for file discovery (issue #1) and CLI discovery wiring.

Discovery walks the current directory recursively for each formatter's
file extensions, skipping hidden directories and the ignored names in
``formattington._tools.IGNORED_DIR_NAMES`` (``__pycache__``,
``node_modules``, ``vendor``, ``venv``, ``dist``, ``build``,
``tools``). When the user passes no explicit file arguments, the CLI
discovers files and appends them to the formatter invocation in
batches; explicit file arguments or informational flags (``--help`` /
``--version``) suppress discovery.

Tests here never invoke real formatter binaries: CLI end-to-end cases
use fake executables on ``PATH`` (POSIX only), like test_integration.
"""

from __future__ import annotations

import os
from pathlib import Path
from unittest.mock import MagicMock

import pytest

import formattington.cli as cli
from formattington._tools import discover_files, iter_chunks
from formattington.cli import main
from tests.conftest import install_fake, read_recorded, set_argv


def _mixed_fixture(root: Path) -> None:
    """Create a mixed-language fixture with ignored directories."""
    (root / "main.c").write_text("int main(void);\n")
    (root / "util.cpp").write_text("int util() { return 1; }\n")
    (root / "run.sh").write_text("#!/bin/sh\necho hi\n")
    (root / "step.bash").write_text("#!/bin/bash\necho hi\n")
    (root / "notes.md").write_text("# notes\n")
    (root / "data.txt").write_text("hello\n")
    sub = root / "sub dir"
    sub.mkdir()
    (sub / "my file.sh").write_text("#!/bin/sh\necho nested\n")
    (sub / "my file.cpp").write_text("int x;\n")
    ignored = root / ".git"
    ignored.mkdir()
    (ignored / "ignored.sh").write_text("#!/bin/sh\necho ignored\n")
    cache = root / "__pycache__"
    cache.mkdir()
    (cache / "ignored.sh").write_text("#!/bin/sh\necho ignored\n")
    modules = root / "node_modules"
    modules.mkdir()
    (modules / "ignored.sh").write_text("#!/bin/sh\necho ignored\n")
    bundled = root / "tools"
    bundled.mkdir()
    (bundled / "ignored.sh").write_text("#!/bin/sh\necho ignored\n")
    hidden = root / ".hidden"
    hidden.mkdir()
    (hidden / "ignored.sh").write_text("#!/bin/sh\necho ignored\n")


class TestDiscoverFiles:
    def test_finds_only_matching_extensions(self, tmp_path):
        _mixed_fixture(tmp_path)
        found = discover_files([".sh", ".bash"], tmp_path)
        assert found == ["run.sh", "step.bash", "sub dir/my file.sh"]
        assert "main.c" not in found
        assert "data.txt" not in found

    def test_finds_c_cpp_extensions(self, tmp_path):
        _mixed_fixture(tmp_path)
        found = discover_files([".c", ".cpp"], tmp_path)
        assert found == ["main.c", "sub dir/my file.cpp", "util.cpp"]

    def test_skips_ignored_directories(self, tmp_path):
        _mixed_fixture(tmp_path)
        found = discover_files([".sh"], tmp_path)
        assert found == ["run.sh", "sub dir/my file.sh"]

    def test_preserves_spaces_as_single_entries(self, tmp_path):
        _mixed_fixture(tmp_path)
        found = discover_files([".sh"], tmp_path)
        assert "sub dir/my file.sh" in found

    def test_empty_directory_returns_empty(self, tmp_path):
        assert discover_files([".sh"], tmp_path) == []

    def test_missing_root_returns_empty(self, tmp_path):
        assert discover_files([".sh"], tmp_path / "does-not-exist") == []

    def test_case_insensitive_extension_match(self, tmp_path):
        (tmp_path / "UPPER.SH").write_text("#!/bin/sh\n")
        assert discover_files([".sh"], tmp_path) == ["UPPER.SH"]

    def test_results_are_sorted(self, tmp_path):
        for name in ("b.sh", "a.sh", "c.sh"):
            (tmp_path / name).write_text("#!/bin/sh\n")
        assert discover_files([".sh"], tmp_path) == ["a.sh", "b.sh", "c.sh"]


class TestIterChunks:
    def test_splits_into_chunks(self):
        assert list(iter_chunks(["a", "b", "c", "d", "e"], 2)) == [
            ["a", "b"],
            ["c", "d"],
            ["e"],
        ]

    def test_single_chunk_when_small(self):
        assert list(iter_chunks(["a"], 256)) == [["a"]]

    def test_empty_items_yield_nothing(self):
        assert list(iter_chunks([], 3)) == []

    def test_nonpositive_size_raises(self):
        with pytest.raises(ValueError, match="chunk size"):
            list(iter_chunks(["a"], 0))


class TestFormatterExtensions:
    def test_shfmt_extensions(self):
        from formattington.shfmt import FILE_EXTENSIONS

        assert FILE_EXTENSIONS == (".sh", ".bash")

    def test_clang_format_extensions(self):
        from formattington.clang_format import FILE_EXTENSIONS

        assert ".c" in FILE_EXTENSIONS
        assert ".cpp" in FILE_EXTENSIONS
        assert ".hpp" in FILE_EXTENSIONS

    def test_cli_extension_map_covers_all_formatters(self):
        assert set(cli.FORMATTER_EXTENSIONS) == set(cli.FORMATTERS)


def _install_discovery_fake(monkeypatch, tmp_path, fake_argv_file, name):
    """Install a PATH fake, pin Darwin, and chdir to the fixture root.

    Single helper covering TOOLS_DIR + FAKE_ARGV_FILE + Darwin pin,
    matching the setup extracted in test_integration. The core
    PATH-fake logic lives in conftest.install_fake; this wrapper adds
    the deterministic Darwin pin.
    """
    install_fake(monkeypatch, tmp_path, fake_argv_file, name)
    monkeypatch.setattr("formattington._tools.platform.system", lambda: "Darwin")
    monkeypatch.setattr("formattington._tools.platform.machine", lambda: "x86_64")


@pytest.mark.skipif(os.name == "nt", reason="shell fakes require POSIX")
class TestCliDiscoveryEndToEnd:
    def test_shfmt_discovers_shell_files_only(
        self, monkeypatch, tmp_path, fake_argv_file
    ):
        _mixed_fixture(tmp_path)
        _install_discovery_fake(monkeypatch, tmp_path, fake_argv_file, "shfmt")
        set_argv(monkeypatch, ["formattington", "shfmt"])
        assert main() == 0
        recorded = read_recorded(fake_argv_file)
        assert recorded == ["-w", "run.sh", "step.bash", "sub dir/my file.sh"]
        # Other extensions untouched: single batch, no .c/.cpp/.md/.txt.
        assert "main.c" not in recorded
        assert "util.cpp" not in recorded

    def test_clang_format_discovers_c_files_only(
        self, monkeypatch, tmp_path, fake_argv_file
    ):
        _mixed_fixture(tmp_path)
        _install_discovery_fake(monkeypatch, tmp_path, fake_argv_file, "clang-format")
        set_argv(monkeypatch, ["formattington", "clang-format"])
        assert main() == 0
        recorded = read_recorded(fake_argv_file)
        assert recorded == ["-i", "main.c", "sub dir/my file.cpp", "util.cpp"]

    def test_explicit_file_args_suppress_discovery(
        self, monkeypatch, tmp_path, fake_argv_file
    ):
        _mixed_fixture(tmp_path)
        _install_discovery_fake(monkeypatch, tmp_path, fake_argv_file, "shfmt")
        set_argv(monkeypatch, ["formattington", "shfmt", "run.sh"])
        assert main() == 0
        assert read_recorded(fake_argv_file) == ["-w", "run.sh"]

    def test_no_matching_files_is_a_noop_batch(
        self, monkeypatch, tmp_path, fake_argv_file
    ):
        # Fixture contains no .sh files (only notes.md / data.txt).
        (tmp_path / "notes.md").write_text("# notes\n")
        _install_discovery_fake(monkeypatch, tmp_path, fake_argv_file, "shfmt")
        set_argv(monkeypatch, ["formattington", "shfmt"])
        assert main() == 0
        # Still one batched invocation with the mode flag only.
        assert read_recorded(fake_argv_file) == ["-w"]


class TestCliDiscoveryWiring:
    def test_discovery_is_appended_to_formatter_args(
        self, monkeypatch, tmp_path
    ):
        _mixed_fixture(tmp_path)
        monkeypatch.chdir(tmp_path)
        mock = MagicMock(return_value=0)
        monkeypatch.setitem(cli.FORMATTERS, "shfmt", mock)
        set_argv(monkeypatch, ["formattington", "--check", "shfmt"])
        assert main() == 0
        mock.assert_called_once_with(
            check=True,
            dry_run=False,
            args=["run.sh", "step.bash", "sub dir/my file.sh"], verbose=False,
        )

    def test_informational_args_skip_discovery(self, monkeypatch, tmp_path):
        _mixed_fixture(tmp_path)
        monkeypatch.chdir(tmp_path)
        mock = MagicMock(return_value=0)
        monkeypatch.setitem(cli.FORMATTERS, "shfmt", mock)
        set_argv(monkeypatch, ["formattington", "shfmt", "--help"])
        assert main() == 0
        mock.assert_called_once_with(check=False, dry_run=False, args=["--help"], verbose=False)

    def test_flags_still_discover(self, monkeypatch, tmp_path):
        _mixed_fixture(tmp_path)
        monkeypatch.chdir(tmp_path)
        mock = MagicMock(return_value=0)
        monkeypatch.setitem(cli.FORMATTERS, "shfmt", mock)
        set_argv(monkeypatch, ["formattington", "shfmt", "--keep-padding", "-i", "2"])
        assert main() == 0
        mock.assert_called_once_with(
            check=False,
            dry_run=False,
            args=[
                "--keep-padding",
                "-i",
                "2",
                "run.sh",
                "step.bash",
                "sub dir/my file.sh",
            ], verbose=False,
        )

    def test_batches_respect_max_files(self, monkeypatch, tmp_path):
        monkeypatch.chdir(tmp_path)
        files = [f"f{i}.sh" for i in range(5)]
        for name in files:
            (tmp_path / name).write_text("#!/bin/sh\n")
        mock = MagicMock(return_value=0)
        monkeypatch.setitem(cli.FORMATTERS, "shfmt", mock)
        monkeypatch.setattr(cli, "MAX_FILES_PER_INVOCATION", 2)
        set_argv(monkeypatch, ["formattington", "shfmt"])
        assert main() == 0
        assert mock.call_count == 3
        seen = [call.kwargs["args"] for call in mock.call_args_list]
        assert seen == [files[0:2], files[2:4], files[4:5]]

    def test_batch_exit_aggregates_last_nonzero(self, monkeypatch, tmp_path):
        monkeypatch.chdir(tmp_path)
        for name in ("a.sh", "b.sh", "c.sh"):
            (tmp_path / name).write_text("#!/bin/sh\n")
        mock = MagicMock(side_effect=[3, 0, 5])
        monkeypatch.setitem(cli.FORMATTERS, "shfmt", mock)
        monkeypatch.setattr(cli, "MAX_FILES_PER_INVOCATION", 1)
        set_argv(monkeypatch, ["formattington", "shfmt"])
        assert main() == 5

    def test_all_discovers_per_formatter(self, monkeypatch, tmp_path):
        _mixed_fixture(tmp_path)
        monkeypatch.chdir(tmp_path)
        mocks = {
            name: MagicMock(return_value=0) for name in cli.FORMATTERS
        }
        original = dict(cli.FORMATTERS)
        cli.FORMATTERS.update(mocks)
        try:
            set_argv(monkeypatch, ["formattington", "all", "--series"])
            assert main() == 0
        finally:
            cli.FORMATTERS.clear()
            cli.FORMATTERS.update(original)
        # shfmt got shell files, clang-format got C files; black/prettier
        # got nothing matching in this fixture (no .py / web files).
        assert mocks["shfmt"].call_args.kwargs["args"] == [
            "run.sh",
            "step.bash",
            "sub dir/my file.sh",
        ]
        assert mocks["clang-format"].call_args.kwargs["args"] == [
            "main.c",
            "sub dir/my file.cpp",
            "util.cpp",
        ]
