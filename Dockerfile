# Dockerfile for Huawei Cloud Ops Skills Sandbox
# Provides isolated environment for running hcloud-skills with CLI and SDK support

FROM ubuntu:22.04

LABEL maintainer="hcloud-skills"
LABEL description="Huawei Cloud Ops Skills Sandbox - CLI + Go SDK + Tools"
LABEL version="1.0.0"

# Prevent interactive prompts during apt install
ENV DEBIAN_FRONTEND=noninteractive

# Set locale
ENV LANG=C.UTF-8
ENV LC_ALL=C.UTF-8

# Install base dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    wget \
    ca-certificates \
    gnupg \
    lsb-release \
    sudo \
    git \
    jq \
    yq \
    bash-completion \
    vim \
    less \
    tree \
    unzip \
    zip \
    net-tools \
    iputils-ping \
    openssh-client \
    python3 \
    python3-pip \
    && rm -rf /var/lib/apt/lists/*

# Install Huawei Cloud CLI (KooCLI)
RUN curl -sSL https://cn-north-4.myhuaweicloud.com/cli/latest/hcloud_install.sh -o /tmp/hcloud_install.sh \
    && bash /tmp/hcloud_install.sh -y \
    && rm /tmp/hcloud_install.sh

# Verify hcloud installation
RUN hcloud version

# Install Go 1.21 for SDK scripts
ENV GO_VERSION=1.21.5
RUN curl -sSL https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz -o /tmp/go.tar.gz \
    && tar -C /usr/local -xzf /tmp/go.tar.gz \
    && rm /tmp/go.tar.gz

# Set Go environment
ENV PATH=$PATH:/usr/local/go/bin:/root/go/bin
ENV GOPATH=/root/go

# Verify Go installation
RUN go version

# Install Go SDK dependencies for Huawei Cloud
RUN go install github.com/huaweicloud/huaweicloud-sdk-go-v3@latest || true

# Create working directory
WORKDIR /workspace/hcloud-skills

# Create directories for credentials and outputs
RUN mkdir -p /workspace/.credentials \
    && mkdir -p /workspace/.outputs \
    && mkdir -p /workspace/logs

# Set up bash profile with useful aliases and functions
RUN echo '# Huawei Cloud Ops Skills Sandbox Profile\n\
export PS1="\\[\\e[1;34m\\]hcloud-skills\\[\\e[0m\\] \\w \\$ "\n\
\n\
# HCloud CLI aliases\n\
alias hc="hcloud"\n\
alias hcv="hcloud version"\n\
alias hcl="hcloud --help"\n\
\n\
# Useful functions\n\
check-env() {\n\
    echo "=== Huawei Cloud Environment ==="\n\
    echo "HW_ACCESS_KEY_ID: ${HW_ACCESS_KEY_ID:-(not set)}"\n\
    echo "HW_SECRET_ACCESS_KEY: ${HW_SECRET_ACCESS_KEY:-(set but hidden)}"\n\
    echo "HW_REGION_ID: ${HW_REGION_ID:-(not set)}"\n\
    echo "HW_PROJECT_ID: ${HW_PROJECT_ID:-(not set)}"\n\
}\n\
\n\
skill-list() {\n\
    find /workspace/hcloud-skills -name "SKILL.md" -type f | sed "s|/workspace/hcloud-skills/||g" | sed "s|/SKILL.md||g"\n\
}\n\
\n\
skill-read() {\n\
    local skill="$1"\n\
    local file="/workspace/hcloud-skills/${skill}/SKILL.md"\n\
    if [ -f "$file" ]; then\n\
        cat "$file"\n\
    else\n\
        echo "Skill not found: $skill"\n\
        echo "Available skills:"\n\
        skill-list\n\
    fi\n\
}\n\
' > /root/.bashrc

# Copy entrypoint script
COPY docker/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD hcloud version || exit 1

# Expose no ports by default (CLI-only tool)
# Skills that need web access (e.g., CloudShell) should use network_mode: host

# Entrypoint with environment validation
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]

# Default command: interactive shell
CMD ["bash"]