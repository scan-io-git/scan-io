#!/bin/sh
# Image-wide dependency overlay for custom Scanio builds.
#
# Runs as root in the runtime stage, after the per-plugin dependency loop and
# while .build-deps (gcc, curl, openssl, musl-dev, ...) are still installed,
# just before they are removed. Anything installed here inherits the image
# cleanup pass. Use it for extra apk packages, pip tools in a venv, or
# downloaded binaries (pin versions and verify checksums, like the trufflehog
# step does). APP_NAME, TARGETOS and TARGETARCH are available as env vars.
#
# Override this file via scripts/custom-build (see: copy-deps-script).
# Default: no-op.
exit 0
