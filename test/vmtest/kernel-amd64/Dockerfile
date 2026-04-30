FROM ubuntu:rolling AS base

ARG LINUX_VERSION
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
	apt-get install -y --no-install-recommends \
		bc \
		bison \
		ca-certificates \
		curl \
		flex \
		gcc \
		libelf-dev \
		libssl-dev \
		make \
		xz-utils \
		zstd && \
	rm -rf /var/lib/apt/lists/*

WORKDIR /root

RUN set -eux; \
	test -n "$LINUX_VERSION"; \
	major="${LINUX_VERSION%%.*}"; \
	curl --fail --location --output linux.tar.xz \
		"https://cdn.kernel.org/pub/linux/kernel/v${major}.x/linux-${LINUX_VERSION}.tar.xz"; \
	mkdir linux; \
	tar -C linux --strip-components=1 -xf linux.tar.xz; \
	rm linux.tar.xz

COPY configs/kernel/bootup-amd64-qemu.fragment linux/.config
COPY scripts/check-kernel-config.sh /usr/local/bin/check-kernel-config
RUN chmod 0444 linux/.config && chmod 0555 /usr/local/bin/check-kernel-config

RUN cd linux && \
	make olddefconfig && \
	check-kernel-config .config && \
	make -j"$(($(nproc) * 2 + 1))"

FROM scratch
COPY --from=base /root/linux/arch/x86_64/boot/bzImage /bzImage
COPY --from=base /root/linux/.config /config_linux.resolved
