FROM docker.io/library/ubuntu:23.10

ENV WINEARCH=win32

RUN DEBIAN_FRONTEND=noninteractive \
    dpkg --add-architecture i386 \
    && apt-get update && apt-get install --no-install-recommends -y \
      ca-certificates \
      apt-transport-https \
      lsb-release \
      curl \
      unzip \
    && mkdir -p /etc/apt/keyrings \
    && curl -L -o /etc/apt/keyrings/winehq-archive.key https://dl.winehq.org/wine-builds/winehq.key \
    && curl -L --output-dir /etc/apt/sources.list.d/ -O "https://dl.winehq.org/wine-builds/ubuntu/dists/$(lsb_release -cs)/winehq-$(lsb_release -cs).sources" \
    && apt-get update && apt-get install --no-install-recommends -y \
      winehq-stable \
    && rm -rf /var/lib/apt/lists/*

RUN curl -L --output-dir / -O https://github.com/Winetricks/winetricks/archive/refs/tags/20230212.tar.gz \
    && tar xa -C / -f 20230212.tar.gz \
    && cp -af /winetricks-20230212/src/winetricks /usr/local/bin/

RUN curl -L -o /wix.zip https://github.com/wixtoolset/wix3/releases/download/wix311rtm/wix311-binaries.zip \
    && unzip -d /usr/local/wix /wix.zip \
    && rm -f /wix.zip

RUN /usr/local/bin/winetricks --unattended dotnet40 \
    && rm -rf /tmp/.wine-* /tmp/winetricks*

COPY build-msi.sh /usr/local/bin/
