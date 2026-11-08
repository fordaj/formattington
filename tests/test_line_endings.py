"""Tests for line-ending helpers."""

from __future__ import annotations

from formattington.line_endings import dos2unix, unix2dos


def test_dos2unix_normalizes(tmp_path):
    path = tmp_path / "script.sh"
    path.write_bytes(b"a\r\nb\r\nc\n")
    dos2unix(path)
    assert path.read_bytes() == b"a\nb\nc\n"


def test_unix2dos_converts(tmp_path):
    path = tmp_path / "script.sh"
    path.write_bytes(b"a\nb\n")
    unix2dos(path)
    assert path.read_bytes() == b"a\r\nb\r\n"


def test_unix2dos_is_idempotent_on_crlf(tmp_path):
    path = tmp_path / "script.sh"
    path.write_bytes(b"a\r\nb\r\n")
    unix2dos(path)
    assert path.read_bytes() == b"a\r\nb\r\n"
