# integer

Minimal, CVE-free container images built daily from [Wolfi](https://github.com/wolfi-dev/os) packages.

Every image is rebuilt nightly, scanned to zero CVEs (CRITICAL through LOW), signed with
[cosign](https://github.com/sigstore/cosign) keyless signing, and attested with a full SBOM.
No base-image debt, no patch lag — just fresh packages from Wolfi's rolling release.

Images are published to `ghcr.io/verity-org` and cover common runtimes and tools across
default, `-dev`, `-fips`, and other variants where applicable.

## Why

Most upstream images carry CVEs that accumulate between releases. Rebuilding from Wolfi
packages gives you the latest security fixes the moment they land, without waiting for
upstream maintainers to cut a new release.
