# SPDX-FileCopyrightText: 2026 Tillitis AB <tillitis.se>
# SPDX-License-Identifier: BSD-2-Clause

from base64 import b64encode
from enum import Enum
from hashlib import blake2s
from pathlib import Path

import nacl.signing as nacl_signing

from .tt.drivers.tkey import TKeyType
from .tt.models.cdi import calc_cdi_bellatrix, calc_cdi_castor_direct

PUBKEY_LEN = 32


class FullUss(Enum):
    NO = False
    YES = True


def derive_expected_pubkey(
    tkey_type: TKeyType, uss_phrase: str | None, full_uss: FullUss | None = None
) -> str:
    def encode_pubkey(pubkey: bytes) -> str:
        assert len(pubkey) == PUBKEY_LEN

        data = b"\x00\x00\x00\x0bssh-ed25519\x00\x00\x00 " + pubkey
        data_b64 = b64encode(data).decode("ascii")

        return "ssh-ed25519 " + data_b64

    def uss_31_byte(uss_phrase: str) -> bytes:
        return blake2s(uss_phrase.encode("utf-8")).digest()[1:] + b"\x00"

    def uss_32_byte(uss_phrase: str) -> bytes:
        return blake2s(uss_phrase.encode("utf-8")).digest()

    if uss_phrase is None:
        uss_digest = None
    elif uss_phrase == "":
        uss_digest = None
    elif tkey_type == TKeyType.CastorPre:
        uss_digest = uss_32_byte(uss_phrase)
    elif full_uss == FullUss.YES:
        uss_digest = uss_32_byte(uss_phrase)
    else:
        uss_digest = uss_31_byte(uss_phrase)

    match tkey_type:
        case TKeyType.Bellatrix | TKeyType.BellatrixUnlocked:
            device_app = Path("../cmd/tkey-ssh-agent/device-app/signer.bin-v1.0.2")
            cdi = calc_cdi_bellatrix(device_app.read_bytes(), uss_digest=uss_digest)
        case TKeyType.CastorPre:
            device_app = Path(
                "../cmd/tkey-ssh-agent/device-app/signer.bin-castor-alpha-1"
            )
            cdi = calc_cdi_castor_direct(device_app.read_bytes(), uss_digest=uss_digest)

    return encode_pubkey(nacl_signing.SigningKey(cdi).verify_key.encode())
