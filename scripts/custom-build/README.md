# Custom Scanio Build

This page describes the available targets and variables in the [`Makefile`](../../scripts/custom-build/Makefile). This [`Makefile`](../../scripts/custom-build/Makefile) supports custom deployments of Scanio, including cases where users have their own versions of Scanio, plugins, and custom rule sets. The deployment process includes cloning the Scanio repository, applying custom rules, building rules, creating Docker images, and pushing them to a registry.

It also supports relabeling the image under a custom binary name and path root (`APP_NAME`), and installing extra image dependencies (a plugin's external scanner, or standalone tools) through an optional `install-deps.sh` overlay.

For the more info, refer to [Makefile Custom Build](../../docs/reference/makefile-custom-build.md) reference.
