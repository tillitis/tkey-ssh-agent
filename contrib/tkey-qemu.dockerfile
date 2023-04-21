
# This is the tag in the https://github.com/tillitis/tillitis-key1 repo with
# the firmware that our TKey/QEMU will run. The published tkey-qemu image will
# have this as part of its name, the tag is then used for versioning the image
# (could be updates to the TKey QEMU machine).
ARG TKEYREPO_TAG=TK1-23.03.1

# This is what we'll actually checkout when building the firmware. It
# really is the firmware as of the TK1-tag above, but a couple of
# commits later where the firmware checksum was committed!
ARG TKEYREPO_TREEISH=444ee3d26c3acf651ff1bbb12023034ccee6ed68

# Using tkey-builder image for building since it has the deps.
FROM ghcr.io/tillitis/tkey-builder:2 AS builder

ARG TKEYREPO_TREEISH

# Cleaning up /usr/local since we will later COPY all from there
RUN rm -rf \
    /usr/local/bin/* \
    /usr/local/pico-sdk \
    /usr/local/repo-commit-* \
    /usr/local/share/icebox \
    /usr/local/share/yosys

RUN git clone -b tk1 --depth=1 https://github.com/tillitis/qemu /src/qemu \
    && mkdir /src/qemu/build
WORKDIR /src/qemu/build
RUN ../configure --target-list=riscv32-softmmu --disable-werror \
    && make -j "$(nproc --ignore=2)" \
    && make install \
    && git >/usr/local/repo-commit-tillitis--qemu describe --all --always --long --dirty

RUN git clone https://github.com/tillitis/tillitis-key1 /src/tkey
WORKDIR /src/tkey/hw/application_fpga
# QEMU needs the .elf, but we build .bin to check sum
RUN git checkout ${TKEYREPO_TREEISH} \
    && make firmware.bin && sha512sum -c firmware.bin.sha512 \
    && make firmware.elf && cp -af firmware.elf firmware-noconsole.elf \
    && make clean \
    && sed -i "s/-DNOCONSOLE//" Makefile \
    && make firmware.elf && cp -af firmware.elf firmware-console.elf \
    && git >/usr/local/repo-commit-tillitis--key1 describe --all --always --long --dirty


# Our QEMU "runtime" image
FROM docker.io/library/ubuntu:22.10

ARG TKEYREPO_TAG

RUN apt-get -qq update -y \
    && DEBIAN_FRONTEND=noninteractive \
       apt-get install -y --no-install-recommends \
               libglib2.0-0 \
               libusb-1.0-0 \
               libpixman-1-0 \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /usr/local/ /usr/local
COPY --from=builder /src/tkey/hw/application_fpga/firmware-noconsole.elf /tkey-firmware-noconsole.elf
COPY --from=builder /src/tkey/hw/application_fpga/firmware-console.elf   /tkey-firmware-console.elf

CMD [ "qemu-system-riscv32" \
    , "-nographic" \
    , "-chardev", "serial,id=chrid,path=/pty-on-host" \
    , "-M", "tk1,fifo=chrid" \
    , "-bios", "/tkey-firmware-noconsole.elf" \
    , "-d", "trace:riscv_trap,guest_errors" \
]

LABEL org.opencontainers.image.description="Tillitis TKey QEMU machine with firmware ${TKEYREPO_TAG}"
