# SPDX-FileCopyrightText: 2026 Tillitis AB <tillitis.se>
# SPDX-License-Identifier: BSD-2-Clause

from pathlib import Path

from .tt.drivers.qemu_tkey import QEmuTKey, QEmuTKeyMachine
from .tt.drivers.tkey import TKey, TKeyType, Udi


def new_qemu_tkey(
    tkey_type: TKeyType, qemu_path: Path, qemu_usb_mux_path: Path, tmp_path: Path
) -> TKey:
    bellatrix_firmware_path = "./tt/data/firmware/firmware-TK1-24.03.bin"
    castor_firmware_path = "./tt/data/firmware/qemu_firmware-260225-main-2ffd46c.elf"
    castor_flash_image_path = Path(
        "./tt/data/flash_images/flash_image-260225-main-2ffd46c.bin"
    )

    match tkey_type:
        case TKeyType.Bellatrix:
            return QEmuTKey(
                tkey_type,
                QEmuTKeyMachine.TK1,
                qemu_path,
                bellatrix_firmware_path,
                flash_image_path=None,
                usb_controller_path=None,
                udi_override=Udi(vid=0x1337, pid=2, rev=2, serial=0x012345),
            )
        case TKeyType.BellatrixUnlocked:
            return QEmuTKey(
                tkey_type,
                QEmuTKeyMachine.TK1,
                qemu_path,
                bellatrix_firmware_path,
                flash_image_path=None,
                usb_controller_path=None,
                udi_override=Udi(vid=16, pid=8, rev=3, serial=0x012345),
            )
        case TKeyType.CastorPre:
            flash_image_path = tmp_path / castor_flash_image_path.name
            flash_image_path.write_bytes(castor_flash_image_path.read_bytes())

            return QEmuTKey(
                tkey_type,
                QEmuTKeyMachine.TK1_CASTOR,
                qemu_path,
                castor_firmware_path,
                flash_image_path,
                qemu_usb_mux_path,
                Udi(vid=0x1337, pid=3, rev=0, serial=0x012345),
            )
