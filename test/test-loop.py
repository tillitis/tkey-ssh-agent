#!/usr/bin/env python

import struct
import serial
import subprocess
from random import randbytes
from tempfile import NamedTemporaryFile
from time import sleep


class TK1:
    def __init__(self, port="/dev/ttyACM0"):
        self.dev = serial.Serial(port, 62500, timeout=0.1)

    # Bootloader command
    def getNameVersion(self):
        """Request the name and version information from the bootloader"""
        cmd = bytearray([0x50, 0x01])
        # print(' '.join(['{:02x}'.format(i) for i in cmd]))
        self.dev.write(cmd)

        rsp = self.dev.read(1 + 32)
        # print(' '.join(['{:02x}'.format(i) for i in rsp]))

        assert rsp[0] == 0x52
        assert rsp[1] == 0x02

        response = {}
        response["name0"] = "".join([chr(i) for i in rsp[2:6]])
        response["name1"] = "".join([chr(i) for i in rsp[6:10]])
        response["version"] = int(rsp[10])
        return response

    # Signer app
    def getPubKey(self):
        cmd = bytearray([0x58, 0x01])
        # print(' '.join(['{:02x}'.format(i) for i in cmd]))
        self.dev.write(cmd)

        rsp = self.dev.read(1 + 128)
        # print(' '.join(['{:02x}'.format(i) for i in rsp]))

        assert rsp[0] == 0x5B
        assert rsp[1] == 0x02
        return rsp[2:34]

    def inSignerApp(self):
        for i in range(0, 2):
            try:
                self.dev.write(bytes(128))
                key = self.getPubKey()
                # print(','.join(['0x{:02x}'.format(i) for i in key]))
                # assert(key == bytearray([
                #    0x67,0xb1,0x46,0x4a,0xa2,0x4f,0x65,0x93,
                #    0xfe,0x67,0x1e,0xc1,0x00,0xf3,0x0e,0x85,
                #    0x8c,0xdf,0x7f,0xbb,0x0b,0x46,0x86,0xbd,
                #    0xf9,0xca,0x47,0xb5,0xc6,0x48,0xba,0x0f
                #    ]))
                return True
            except Exception:
                pass

        return False

    def inBootloader(self):
        for i in range(0, 2):
            try:
                self.dev.write(bytes(128))
                response = self.getNameVersion()
                # print(response, len(response['name1']))
                # assert(response['name0'] == 'tk1 ')
                # assert(response['name1'] == 'mkdf')
                # assert(response['version'] == 4)
                return True
            except Exception:
                pass

        return False


def probe_state():
    """Probe the TK1 to determine if it is running the bootloader or signer"""
    try:
        key = TK1()

        # First, try to read the public key
        # If this is successful, the signer app is loaded
        if key.inSignerApp():
            return "signer"

        if key.inBootloader():
            return "bootloader"
    except Exception:
        pass

    return "unknown"


def load_signer_app():
    try:
        result = subprocess.run(
            ["../tkey-runapp", "--port", "/dev/ttyACM0", "../apps/signer/app.bin"],
            timeout=10,
        )
    except subprocess.TimeoutExpired:
        print("loader process timeout")


def do_signature():
    msgf = NamedTemporaryFile()
    msgf.write(randbytes(128))
    msgf.flush()

    try:
        result = subprocess.run(
            ["../tkey-sign", "--port", "/dev/ttyACM0", msgf.name], timeout=1
        )
    except subprocess.TimeoutExpired:
        print("signature process timeout")


def main():
    stats = {"restarts": 0, "signatures": 0, "disconnects": 0}

    while True:
        state = probe_state()

        print("Detected key in state: " + state)
        if state == "bootloader":
            load_signer_app()
            sleep(2)  # Give time for the app to start
            stats["restarts"] += 1
        elif state == "signer":
            do_signature()
            stats["signatures"] += 1
        else:
            print("Device in unknown state: reconnecting")
            stats["disconnects"] += 1
            sleep(1)

        print(stats)


if __name__ == "__main__":
    main()
