"""Tests for the package export surface (issue #38)."""

import formattington


def test_all_names_importable():
    assert formattington.__all__ == [
        "shfmt",
        "clang_format",
        "black",
        "prettier",
        "powershell",
        "taplo",
        "gofmt",
        "goimports",
    ]
    for name in (
        "shfmt",
        "clang_format",
        "black",
        "prettier",
        "powershell",
        "taplo",
        "gofmt",
        "goimports",
    ):
        assert callable(getattr(formattington, name))
