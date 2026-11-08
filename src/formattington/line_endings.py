from pathlib import Path


def dos2unix(path: Path) -> None:
    data = path.read_bytes()
    data = data.replace(b"\r\n", b"\n")
    path.write_bytes(data)


def unix2dos(path: Path) -> None:
    data = path.read_bytes()
    data = data.replace(b"\r\n", b"\n").replace(b"\n", b"\r\n")
    path.write_bytes(data)