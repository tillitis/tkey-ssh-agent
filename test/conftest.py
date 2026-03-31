# SPDX-FileCopyrightText: 2026 Tillitis AB <tillitis.se>
# SPDX-License-Identifier: BSD-2-Clause

from pathlib import Path
from typing import Iterator

import pytest

from .device import new_qemu_tkey
from .tt.drivers.tkey import TKey, TKeyType


@pytest.fixture(
    params=[TKeyType.Bellatrix, TKeyType.BellatrixUnlocked, TKeyType.CastorPre],
)
def tkey(
    request: pytest.FixtureRequest,
    qemu_path: Path,
    qemu_usb_mux_path: Path,
    tmp_path: Path,
) -> Iterator[TKey]:
    _tkey = new_qemu_tkey(request.param, qemu_path, qemu_usb_mux_path, tmp_path)
    _tkey.insert()
    yield _tkey
    _tkey.eject()
