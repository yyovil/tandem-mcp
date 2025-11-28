FROM kalilinux/kali-last-release AS base

# to say yes to all.
ENV DEBIAN_FRONTEND=noninteractive

# go get the headless metapackage.
RUN apt-get update && apt-get install -y --no-install-recommends kali-linux-headless

# Install freeze for terminal screenshot capture (auto-detect architecture)
RUN ARCH=$(dpkg --print-architecture) && \
    curl -fsSL "https://github.com/charmbracelet/freeze/releases/download/v0.2.2/freeze_0.2.2_${ARCH}.deb" -o /tmp/freeze.deb && \
    dpkg -i /tmp/freeze.deb && \
    rm /tmp/freeze.deb

# Set simple prompt for command tracking (append to bashrc so it overrides Kali defaults)
RUN echo 'export PS1="KALI[\w]# "' >> /root/.bashrc

COPY capture_cmds.sh /usr/local/bin/capture_cmds.sh
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/capture_cmds.sh /usr/local/bin/entrypoint.sh

USER root
WORKDIR /home/pentests

CMD ["/usr/local/bin/entrypoint.sh"]