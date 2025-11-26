FROM kalilinux/kali-rolling:latest

# Update and install required packages
RUN apt-get update && apt-get install -y --no-install-recommends \
    tmux \
    curl \
    bash \
    ca-certificates \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Install freeze from charmbracelet
RUN curl -fsSL https://github.com/charmbracelet/freeze/releases/download/v0.2.2/freeze_0.2.2_Linux_x86_64.tar.gz | tar -xzf - -C /usr/local/bin freeze

# Copy the entrypoint script
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Set working directory
WORKDIR /root

# Set default entrypoint
ENTRYPOINT ["/entrypoint.sh"]
