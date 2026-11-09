"""Integration-style tests with fake executables.

These tests never invoke the bundled or real formatter binaries. Instead they
point tool discovery at temporary directories containing stub scripts (or PATH
stubs) and prove that arguments — including paths with spaces — arrive as
correct argv batches in a single subprocess invocation.

On POSIX the fakes are shell scripts executed for real. On Windows they are
skipped via @pytest.mark.skipif; the mocked-subprocess tests in
test_formatters cover the same argv contracts there.
"""

from __future__ import annotations

import os
from pathlib import Path

import pytest

import formattington._tools as tools
from formattington.clang_format import clang_format
from formattington.format_python import black
from formattington.gofmt import gofmt
from formattington.goimports import goimports
from formattington.prettier import prettier
from formattington.shfmt import shfmt
from tests.conftest import (
    make_fake_executable,
    make_tools_layout,
    read_recorded,
    set_argv,
)


def _pin_darwin(monkeypatch) -> None:
    monkeypatch.setattr("formattington._tools.platform.system", lambda: "Darwin")
    monkeypatch.setattr("formattington._tools.platform.machine", lambda: "x86_64")


def _install_bundled(
    monkeypatch,
    tmp_path: Path,
    fake_argv_file: Path,
    name: str,
    version: str,
    platform_dir: str,
    exe_name: str,
) -> None:
    tools_dir = tmp_path / "tools"
    make_tools_layout(tools_dir, name, version, platform_dir, exe_name)
    monkeypatch.setattr(tools, "TOOLS_DIR", tools_dir)
    monkeypatch.setenv("FAKE_ARGV_FILE", str(fake_argv_file))
    _pin_darwin(monkeypatch)


def _install_path_fakes(
    monkeypatch,
    tmp_path: Path,
    fake_argv_file: Path,
    *exe_names: str,
) -> None:
    empty_tools = tmp_path / "empty-tools"
    empty_tools.mkdir()
    monkeypatch.setattr(tools, "TOOLS_DIR", empty_tools)
    bin_dir = tmp_path / "bin"
    for exe_name in exe_names:
        make_fake_executable(bin_dir, exe_name)
    monkeypatch.setenv("PATH", f"{bin_dir}{os.pathsep}{os.environ.get('PATH', '')}")
    monkeypatch.setenv("FAKE_ARGV_FILE", str(fake_argv_file))
    _pin_darwin(monkeypatch)


@pytest.mark.skipif(os.name == "nt", reason="shell fakes require POSIX")
class TestBundledFakeExecutables:
    def test_shfmt_finds_bundled_fake_and_records_write_mode(
        self, monkeypatch, tmp_path, fake_argv_file
    ):
        _install_bundled(
            monkeypatch, tmp_path, fake_argv_file, "shfmt", "3.14.1", "macos-x86_64", "shfmt"
        )
        assert shfmt() == 0
        # One bare "-w" argument: the mode flag and nothing else.
        assert read_recorded(fake_argv_file) == ["-w"]

    def test_shfmt_batches_many_paths_in_one_invocation(
        self, monkeypatch, tmp_path, fake_argv_file
    ):
        _install_bundled(
            monkeypatch, tmp_path, fake_argv_file, "shfmt", "3.14.1", "macos-x86_64", "shfmt"
        )
        paths = [
            "a.sh",
            "b with spaces.sh",
            "very/long/path/to/c.sh",
            "d.sh",
        ]
        assert shfmt(args=paths) == 0
        # A single invocation carried every path (batch, not per-file).
        # The fake appends across invocations, so len(recorded) == len(paths)+1
        # proves one call happened; spaces stayed intact as single entries.
        recorded = read_recorded(fake_argv_file)
        assert recorded[0] == "-w"
        assert recorded[1:] == paths
        assert "b with spaces.sh" in recorded

    def test_shfmt_check_mode_with_fake(self, monkeypatch, tmp_path, fake_argv_file):
        _install_bundled(
            monkeypatch, tmp_path, fake_argv_file, "shfmt", "3.14.1", "macos-x86_64", "shfmt"
        )
        assert shfmt(check=True) == 0
        assert read_recorded(fake_argv_file) == ["-l"]

    def test_clang_format_batches_paths_with_spaces(
        self, monkeypatch, tmp_path, fake_argv_file
    ):
        _install_bundled(
            monkeypatch,
            tmp_path,
            fake_argv_file,
            "clang-format",
            "23.1.3",
            "macos-x86_64",
            "clang-format",
        )
        assert clang_format(args=["my file.cpp", "another file.cpp"]) == 0
        recorded = read_recorded(fake_argv_file)
        assert recorded == ["-i", "my file.cpp", "another file.cpp"]

    def test_run_tool_batch_chunking_is_a_single_call(
        self, monkeypatch, tmp_path, fake_argv_file
    ):
        """run_tool sends all arguments in one subprocess.run call.

        This is the batching contract that future file discovery relies on:
        many paths in, one invocation. The fake executable records exactly one
        argv batch, so a second read (via integration with shfmt) would double
        the lines; here we assert each path appears exactly once.
        """
        _install_bundled(
            monkeypatch, tmp_path, fake_argv_file, "shfmt", "3.14.1", "macos-x86_64", "shfmt"
        )
        exe = tools.find_tool("shfmt")
        assert tools.run_tool(exe, ["-w", "a.sh", "b with spaces.sh"]) == 0
        recorded = read_recorded(fake_argv_file)
        assert recorded == ["-w", "a.sh", "b with spaces.sh"]
        # Each path counted once => single batched invocation.
        assert recorded.count("b with spaces.sh") == 1


@pytest.mark.skipif(os.name == "nt", reason="shell fakes require POSIX")
class TestPathStubs:
    def test_falls_back_to_path_stub(self, monkeypatch, tmp_path, fake_argv_file):
        _install_path_fakes(monkeypatch, tmp_path, fake_argv_file, "shfmt")
        assert shfmt(args=["file with spaces.sh"]) == 0
        assert read_recorded(fake_argv_file) == ["-w", "file with spaces.sh"]

    def test_cli_uses_path_stub_end_to_end(self, monkeypatch, tmp_path, fake_argv_file):
        _install_path_fakes(monkeypatch, tmp_path, fake_argv_file, "shfmt", "clang-format")
        set_argv(monkeypatch, ["formattington", "--check", "shfmt"])
        # Run from an empty directory so automatic discovery is hermetic.
        monkeypatch.chdir(tmp_path)
        from formattington.cli import main

        assert main() == 0
        # --check before the formatter maps to shfmt's "-l" batch.
        assert read_recorded(fake_argv_file) == ["-l"]

    def test_black_batches_paths_with_spaces(
        self, monkeypatch, tmp_path, fake_argv_file
    ):
        _install_path_fakes(monkeypatch, tmp_path, fake_argv_file, "black")
        assert black(args=["my file.py", "another file.py"]) == 0
        assert read_recorded(fake_argv_file) == ["my file.py", "another file.py"]

    def test_black_check_mode(self, monkeypatch, tmp_path, fake_argv_file):
        _install_path_fakes(monkeypatch, tmp_path, fake_argv_file, "black")
        assert black(check=True, args=["file with spaces.py"]) == 0
        assert read_recorded(fake_argv_file) == ["--check", "file with spaces.py"]

    def test_black_dry_run_mode(self, monkeypatch, tmp_path, fake_argv_file):
        _install_path_fakes(monkeypatch, tmp_path, fake_argv_file, "black")
        assert black(dry_run=True, args=["file.py"]) == 0
        assert read_recorded(fake_argv_file) == ["--diff", "file.py"]

    def test_black_via_cli_end_to_end(self, monkeypatch, tmp_path, fake_argv_file):
        _install_path_fakes(monkeypatch, tmp_path, fake_argv_file, "black")
        set_argv(monkeypatch, ["formattington", "--check", "black", "my file.py"])
        from formattington.cli import main

        assert main() == 0
        assert read_recorded(fake_argv_file) == ["--check", "my file.py"]

    def test_prettier_batches_paths_with_spaces(
        self, monkeypatch, tmp_path, fake_argv_file
    ):
        _install_path_fakes(monkeypatch, tmp_path, fake_argv_file, "prettier")
        assert prettier(args=["my file.js", "another file.json"]) == 0
        assert read_recorded(fake_argv_file) == [
            "--write",
            "my file.js",
            "another file.json",
        ]

    def test_prettier_check_mode(self, monkeypatch, tmp_path, fake_argv_file):
        _install_path_fakes(monkeypatch, tmp_path, fake_argv_file, "prettier")
        assert prettier(check=True, args=["file with spaces.js"]) == 0
        assert read_recorded(fake_argv_file) == ["--check", "file with spaces.js"]

    def test_prettier_dry_run_mode(self, monkeypatch, tmp_path, fake_argv_file):
        _install_path_fakes(monkeypatch, tmp_path, fake_argv_file, "prettier")
        assert prettier(dry_run=True, args=["file.js"]) == 0
        assert read_recorded(fake_argv_file) == ["--list-different", "file.js"]

    def test_prettier_via_cli_end_to_end(self, monkeypatch, tmp_path, fake_argv_file):
        _install_path_fakes(monkeypatch, tmp_path, fake_argv_file, "prettier")
        set_argv(monkeypatch, ["formattington", "--check", "prettier", "my file.js"])
        from formattington.cli import main

        assert main() == 0
        assert read_recorded(fake_argv_file) == ["--check", "my file.js"]

    def test_prettier_forwarded_check_via_cli(self, monkeypatch, tmp_path, fake_argv_file):
        _install_path_fakes(monkeypatch, tmp_path, fake_argv_file, "prettier")
        set_argv(monkeypatch, ["formattington", "prettier", "--check"])
        # Discovery is automatic when no files are forwarded, so run from
        # an empty directory to keep the argv assertion hermetic.
        monkeypatch.chdir(tmp_path)
        from formattington.cli import main

        assert main() == 0
        # Forwarded --check omits the default --write.
        assert read_recorded(fake_argv_file) == ["--check"]

    def test_gofmt_batches_paths_with_spaces(
        self, monkeypatch, tmp_path, fake_argv_file
    ):
        _install_path_fakes(monkeypatch, tmp_path, fake_argv_file, "gofmt")
        assert gofmt(args=["my file.go", "another file.go"]) == 0
        assert read_recorded(fake_argv_file) == [
            "-w",
            "my file.go",
            "another file.go",
        ]

    def test_gofmt_check_mode(self, monkeypatch, tmp_path, fake_argv_file):
        _install_path_fakes(monkeypatch, tmp_path, fake_argv_file, "gofmt")
        assert gofmt(check=True, args=["file with spaces.go"]) == 0
        assert read_recorded(fake_argv_file) == ["-l", "file with spaces.go"]

    def test_gofmt_dry_run_mode(self, monkeypatch, tmp_path, fake_argv_file):
        _install_path_fakes(monkeypatch, tmp_path, fake_argv_file, "gofmt")
        assert gofmt(dry_run=True, args=["file.go"]) == 0
        assert read_recorded(fake_argv_file) == ["-d", "file.go"]

    def test_gofmt_via_cli_end_to_end(self, monkeypatch, tmp_path, fake_argv_file):
        _install_path_fakes(monkeypatch, tmp_path, fake_argv_file, "gofmt")
        set_argv(monkeypatch, ["formattington", "--check", "gofmt", "my file.go"])
        from formattington.cli import main

        assert main() == 0
        assert read_recorded(fake_argv_file) == ["-l", "my file.go"]

    def test_goimports_batches_paths_with_spaces(
        self, monkeypatch, tmp_path, fake_argv_file
    ):
        _install_path_fakes(monkeypatch, tmp_path, fake_argv_file, "goimports")
        assert goimports(args=["my file.go", "another file.go"]) == 0
        assert read_recorded(fake_argv_file) == [
            "-w",
            "my file.go",
            "another file.go",
        ]

    def test_goimports_check_mode(self, monkeypatch, tmp_path, fake_argv_file):
        _install_path_fakes(monkeypatch, tmp_path, fake_argv_file, "goimports")
        assert goimports(check=True, args=["file with spaces.go"]) == 0
        assert read_recorded(fake_argv_file) == ["-l", "file with spaces.go"]

    def test_goimports_dry_run_mode(self, monkeypatch, tmp_path, fake_argv_file):
        _install_path_fakes(monkeypatch, tmp_path, fake_argv_file, "goimports")
        assert goimports(dry_run=True, args=["file.go"]) == 0
        assert read_recorded(fake_argv_file) == ["-d", "file.go"]

    def test_goimports_via_cli_end_to_end(self, monkeypatch, tmp_path, fake_argv_file):
        _install_path_fakes(monkeypatch, tmp_path, fake_argv_file, "goimports")
        set_argv(monkeypatch, ["formattington", "--check", "goimports", "my file.go"])
        from formattington.cli import main

        assert main() == 0
        assert read_recorded(fake_argv_file) == ["-l", "my file.go"]


class TestFileDiscoveryContract:
    """Discovery-agnostic checks that hold on every platform.

    The current implementation has no separate per-file discovery module;
    formatters receive whole argument batches from the CLI. These tests lock
    in the contract future discovery must preserve: one subprocess call per
    formatter, arguments as discrete argv entries, spaces intact.
    """

    def test_shfmt_single_call_for_many_paths(self):
        from unittest.mock import patch

        paths = [f"dir/file {i}.sh" for i in range(5)]
        calls = []

        def fake_run_tool(executable, arguments, *, capture_output=False):
            calls.append((executable, list(arguments)))
            return 0

        with patch(
            "formattington.shfmt.find_tool", return_value="/fake/shfmt"
        ), patch("formattington.shfmt.run_tool", side_effect=fake_run_tool):
            assert shfmt(args=paths) == 0
        assert len(calls) == 1
        _, arguments = calls[0]
        assert arguments == ["-w", *paths]

    def test_clang_format_single_call_for_many_paths(self):
        from unittest.mock import patch

        paths = [f"dir/file {i}.cpp" for i in range(5)]
        calls = []

        def fake_run(cmd, **kwargs):
            calls.append(list(cmd))
            m = type("R", (), {"returncode": 0})()
            return m

        with patch(
            "formattington.clang_format.find_tool", return_value="/fake/clang-format"
        ), patch("formattington.clang_format.subprocess.run", side_effect=fake_run):
            assert clang_format(args=paths) == 0
        assert len(calls) == 1
        assert calls[0] == ["/fake/clang-format", "-i", *paths]

    def test_black_single_call_for_many_paths(self):
        from unittest.mock import patch

        paths = [f"dir/file {i}.py" for i in range(5)]
        calls = []

        def fake_run(cmd, **kwargs):
            calls.append(list(cmd))
            m = type("R", (), {"returncode": 0})()
            return m

        with patch(
            "formattington.format_python.find_tool", return_value="/fake/black"
        ), patch("formattington.format_python.subprocess.run", side_effect=fake_run):
            assert black(args=paths) == 0
        assert len(calls) == 1
        assert calls[0] == ["/fake/black", *paths]

    def test_prettier_single_call_for_many_paths(self):
        from unittest.mock import patch

        paths = [f"dir/file {i}.js" for i in range(5)]
        calls = []

        def fake_run(cmd, **kwargs):
            calls.append(list(cmd))
            m = type("R", (), {"returncode": 0})()
            return m

        with patch(
            "formattington.prettier.find_tool", return_value="/fake/prettier"
        ), patch("formattington.prettier.subprocess.run", side_effect=fake_run):
            assert prettier(args=paths) == 0
        assert len(calls) == 1
        assert calls[0] == ["/fake/prettier", "--write", *paths]

    def test_gofmt_single_call_for_many_paths(self):
        from unittest.mock import patch

        paths = [f"dir/file {i}.go" for i in range(5)]
        calls = []

        def fake_run(cmd, **kwargs):
            calls.append(list(cmd))
            m = type("R", (), {"returncode": 0})()
            return m

        with patch(
            "formattington.gofmt.find_tool", return_value="/fake/gofmt"
        ), patch("formattington.gofmt.subprocess.run", side_effect=fake_run):
            assert gofmt(args=paths) == 0
        assert len(calls) == 1
        assert calls[0] == ["/fake/gofmt", "-w", *paths]

    def test_goimports_single_call_for_many_paths(self):
        from unittest.mock import patch

        paths = [f"dir/file {i}.go" for i in range(5)]
        calls = []

        def fake_run(cmd, **kwargs):
            calls.append(list(cmd))
            m = type("R", (), {"returncode": 0})()
            return m

        with patch(
            "formattington.goimports.find_tool", return_value="/fake/goimports"
        ), patch("formattington.goimports.subprocess.run", side_effect=fake_run):
            assert goimports(args=paths) == 0
        assert len(calls) == 1
        assert calls[0] == ["/fake/goimports", "-w", *paths]
