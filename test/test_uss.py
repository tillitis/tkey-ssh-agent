# SPDX-FileCopyrightText: 2026 Tillitis AB <tillitis.se>
# SPDX-License-Identifier: BSD-2-Clause

from pathlib import Path

import pytest

from .keys import FullUss, derive_expected_pubkey
from .tt.drivers.tkey import TKey
from .tt.drivers.tkey_ssh_agent import TKeySshAgent

APP_PATH = Path("../tkey-ssh-agent").absolute()
FAKE_PIN_ENTRY_PATH = Path("./tt/tools/fake_pinentry.py").absolute()

USS_PHRASES = [
    "",
    pytest.param("adl", marks=pytest.mark.issue("GHSA-4w7r-3222-8h6v")),
    "a uss phrase",
    "a very long phrase..." * 10,
]


def test_should_show_expected_pubkey_when_not_using_uss(tkey: TKey) -> None:
    tkey_ssh_agent = TKeySshAgent(APP_PATH, tkey)

    pubkey = tkey_ssh_agent.show_pubkey()

    assert pubkey == derive_expected_pubkey(tkey.type(), uss_phrase=None)


@pytest.mark.parametrize("uss_phrase", USS_PHRASES)
@pytest.mark.parametrize("force_full_uss", [FullUss.NO, FullUss.YES])
def test_should_show_expected_pubkey_when_using_uss(
    uss_phrase: str,
    force_full_uss: FullUss,
    tkey: TKey,
    tmp_path: Path,
) -> None:
    tkey_ssh_agent = TKeySshAgent(APP_PATH, tkey, cwd=tmp_path)
    (tmp_path / "fake-pinentry-pin").write_text(uss_phrase)

    args = ["--uss", "--pinentry", str(FAKE_PIN_ENTRY_PATH)]
    if force_full_uss == FullUss.YES:
        args.append("--force-full-uss")

    pubkey = tkey_ssh_agent.show_pubkey(args)

    assert pubkey == derive_expected_pubkey(tkey.type(), uss_phrase, force_full_uss)


@pytest.mark.parametrize("uss_phrase", USS_PHRASES)
@pytest.mark.parametrize("force_full_uss", [FullUss.NO, FullUss.YES])
def test_should_show_expected_pubkey_when_using_uss_file(
    uss_phrase: str,
    force_full_uss: FullUss,
    tkey: TKey,
    tmp_path: Path,
) -> None:
    tkey_ssh_agent = TKeySshAgent(APP_PATH, tkey, cwd=tmp_path)
    uss_file_path = tmp_path / "uss-file.txt"
    uss_file_path.write_text(uss_phrase)

    args = ["--uss-file", str(uss_file_path)]
    if force_full_uss == FullUss.YES:
        args.append("--force-full-uss")

    pubkey = tkey_ssh_agent.show_pubkey(args)

    assert pubkey == derive_expected_pubkey(tkey.type(), uss_phrase, force_full_uss)
