import struct


class Reader:
    def __init__(self, fh):
        self.file = fh

    def __iter__(self):
        while True:
            try:
                len_bytes = self.readexactly(8)
            except EOFError:
                return
            (len,) = struct.unpack(">Q", len_bytes)
            yield self.readexactly(len)

    def close(self):
        self.file.close()

    def readexactly(self, n):
        buf = b""
        while len(buf) < n:
            bs = self.file.read(n - len(buf))
            if len(bs) == 0:
                if len(buf) == 0:
                    raise EOFError
                raise ValueError(
                    f"Partial read, got {len(buf)+len(bs)} bytes, wanted {n}"
                )
            buf += bs
        return buf

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        self.close()


class Writer:
    def __init__(self, fh):
        self.file = fh

    def write(self, bytes):
        len_bytes = struct.pack(">Q", len(bytes))
        assert len(len_bytes) == 8
        self.file.write(len_bytes)
        self.file.write(bytes)

    def close(self):
        self.file.close()

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        self.close()
