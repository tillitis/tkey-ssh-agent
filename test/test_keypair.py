# SPDX-FileCopyrightText: 2026 Tillitis AB <tillitis.se>
# SPDX-License-Identifier: BSD-2-Clause

import os
import platform
import shutil
from pathlib import Path

import pytest

from .keys import FullUss, derive_expected_pubkey
from .tt.drivers.ssh_keygen import SshKeygen
from .tt.drivers.tkey import TKey
from .tt.drivers.tkey_ssh_agent import TKeySshAgent
from .tt.utils.path import wait_until_path_exists

APP_PATH = Path("../tkey-ssh-agent").absolute()
FAKE_PIN_ENTRY_PATH = Path("./tt/tools/fake_pinentry.py").absolute()

USS_PHRASES = [
    "",
    pytest.param("adl", marks=pytest.mark.issue("GHSA-4w7r-3222-8h6v")),
    "a uss phrase",
    "a very long phrase..." * 10,
]


@pytest.fixture
def os_env_with_fake_pinentry_in_path(tmp_path: Path) -> dict[str, str]:
    env = os.environ.copy()
    bin_dir = tmp_path / "fake-pinentry"
    bin_dir.mkdir()
    shutil.copy2(FAKE_PIN_ENTRY_PATH, bin_dir / "pinentry")

    env["PATH"] = f"{str(bin_dir)}{os.pathsep}{env['PATH']}"

    return env


def test_should_show_expected_pubkey_when_not_using_uss(tkey: TKey) -> None:
    tkey_ssh_agent = TKeySshAgent(APP_PATH, tkey)

    pubkey = tkey_ssh_agent.show_pubkey()

    assert pubkey == derive_expected_pubkey(tkey.type(), uss_phrase=None)


@pytest.mark.touch
def test_should_sign_with_expected_key_when_not_using_uss(
    tkey: TKey, tmp_path: Path
) -> None:
    expected_pubkey = derive_expected_pubkey(tkey.type(), uss_phrase=None)

    sock_path = tmp_path / "sock"
    args = ["--agent-path", str(sock_path)]
    with TKeySshAgent(APP_PATH, tkey, extra_args=args, cwd=tmp_path):
        wait_until_path_exists(sock_path, timeout=2)

        signature = SshKeygen.sign("a message", expected_pubkey, sock_path)
        SshKeygen.verify("a message", signature, expected_pubkey)


@pytest.mark.skipif(
    platform.system() in ["Darwin", "Windows"],
    reason="test not implemented for macOS or Windows where pinentry is not the default"
    " USS prompt",
)
@pytest.mark.parametrize("uss_phrase", USS_PHRASES)
@pytest.mark.parametrize("force_full_uss", [FullUss.NO, FullUss.YES])
def test_should_show_expected_pubkey_when_using_uss_from_default_pinentry(
    uss_phrase: str,
    force_full_uss: FullUss,
    tkey: TKey,
    tmp_path: Path,
    os_env_with_fake_pinentry_in_path: dict[str, str],
) -> None:
    (tmp_path / "fake-pinentry-pin").write_text(uss_phrase)
    tkey_ssh_agent = TKeySshAgent(
        APP_PATH, tkey, cwd=tmp_path, env=os_env_with_fake_pinentry_in_path
    )

    args = ["--uss"]
    if force_full_uss == FullUss.YES:
        args.append("--force-full-uss")

    pubkey = tkey_ssh_agent.show_pubkey(args)

    assert pubkey == derive_expected_pubkey(tkey.type(), uss_phrase, force_full_uss)


@pytest.mark.skipif(
    platform.system() in ["Darwin", "Windows"],
    reason="test not implemented for macOS or Windows where pinentry is not the default"
    " USS prompt",
)
@pytest.mark.touch
@pytest.mark.parametrize("uss_phrase", USS_PHRASES)
@pytest.mark.parametrize("force_full_uss", [FullUss.NO, FullUss.YES])
def test_should_sign_with_expected_key_when_using_uss_from_default_pinentry(
    uss_phrase: str,
    force_full_uss: FullUss,
    tkey: TKey,
    tmp_path: Path,
    os_env_with_fake_pinentry_in_path: dict[str, str],
) -> None:
    (tmp_path / "fake-pinentry-pin").write_text(uss_phrase)
    sock_path = tmp_path / "sock"

    args = ["--agent-path", str(sock_path)]
    args += ["--uss"]
    if force_full_uss == FullUss.YES:
        args.append("--force-full-uss")

    with TKeySshAgent(
        APP_PATH,
        tkey,
        extra_args=args,
        cwd=tmp_path,
        env=os_env_with_fake_pinentry_in_path,
    ):
        wait_until_path_exists(sock_path, timeout=2)

        expected_pubkey = derive_expected_pubkey(
            tkey.type(), uss_phrase, force_full_uss
        )
        signature = SshKeygen.sign("a message", expected_pubkey, sock_path)
        SshKeygen.verify("a message", signature, expected_pubkey)


@pytest.mark.parametrize("uss_phrase", USS_PHRASES)
@pytest.mark.parametrize("force_full_uss", [FullUss.NO, FullUss.YES])
def test_should_show_expected_pubkey_when_using_uss_from_specific_pinentry(
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


@pytest.mark.touch
@pytest.mark.parametrize("uss_phrase", USS_PHRASES)
@pytest.mark.parametrize("force_full_uss", [FullUss.NO, FullUss.YES])
def test_should_sign_with_expected_key_when_using_uss_from_specific_pinentry(
    uss_phrase: str,
    force_full_uss: FullUss,
    tkey: TKey,
    tmp_path: Path,
) -> None:
    (tmp_path / "fake-pinentry-pin").write_text(uss_phrase)
    sock_path = tmp_path / "sock"

    args = ["--agent-path", str(sock_path)]
    args += ["--uss", "--pinentry", str(FAKE_PIN_ENTRY_PATH)]
    if force_full_uss == FullUss.YES:
        args.append("--force-full-uss")

    with TKeySshAgent(APP_PATH, tkey, extra_args=args, cwd=tmp_path):
        wait_until_path_exists(sock_path, timeout=2)

        expected_pubkey = derive_expected_pubkey(
            tkey.type(), uss_phrase, force_full_uss
        )
        signature = SshKeygen.sign("a message", expected_pubkey, sock_path)
        SshKeygen.verify("a message", signature, expected_pubkey)


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


@pytest.mark.touch
@pytest.mark.parametrize("uss_phrase", USS_PHRASES)
@pytest.mark.parametrize("force_full_uss", [FullUss.NO, FullUss.YES])
def test_should_sign_with_expected_key_when_using_uss_file(
    uss_phrase: str,
    force_full_uss: FullUss,
    tkey: TKey,
    tmp_path: Path,
) -> None:
    uss_file_path = tmp_path / "uss-file.txt"
    uss_file_path.write_text(uss_phrase)
    sock_path = tmp_path / "sock"

    args = ["--agent-path", str(sock_path)]
    args += ["--uss-file", str(uss_file_path)]
    if force_full_uss == FullUss.YES:
        args.append("--force-full-uss")

    with TKeySshAgent(APP_PATH, tkey, extra_args=args, cwd=tmp_path):
        wait_until_path_exists(sock_path, timeout=2)

        expected_pubkey = derive_expected_pubkey(
            tkey.type(), uss_phrase, force_full_uss
        )
        signature = SshKeygen.sign("a message", expected_pubkey, sock_path)
        SshKeygen.verify("a message", signature, expected_pubkey)
