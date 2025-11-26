FROM kalilinux/kali-rolling:latest

# Freeze tool version (from charmbracelet)
ARG FREEZE_VERSION=0.2.2

# Update and install required packages
RUN apt-get update && apt-get install -y --no-install-recommends \
    tmux \
    curl \
    bash \
    ca-certificates \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Install freeze from charmbracelet with checksum verification
RUN curl -fsSL "https://github.com/charmbracelet/freeze/releases/download/v${FREEZE_VERSION}/freeze_${FREEZE_VERSION}_Linux_x86_64.tar.gz" -o /tmp/freeze.tar.gz && \
    curl -fsSL "https://github.com/charmbracelet/freeze/releases/download/v${FREEZE_VERSION}/checksums.txt" -o /tmp/checksums.txt && \
    cd /tmp && grep "freeze_${FREEZE_VERSION}_Linux_x86_64.tar.gz" checksums.txt | sha256sum -c - && \
    tar -xzf freeze.tar.gz -C /usr/local/bin freeze && \
    rm -f /tmp/freeze.tar.gz /tmp/checksums.txt

# Copy the entrypoint script
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Set working directory
WORKDIR /root

# Set default entrypoint
ENTRYPOINT ["/entrypoint.sh"]
