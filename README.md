# Legolas

An alternative implementation of [patchelf](https://github.com/NixOS/patchelf). It re-links binaries with other RPATHs, thus, allows them to not depend on the host system's `/usr/bin`, and `/usr/lib`.

This is mainly meant to be a Proof Of Concept to test out if this actually could work in [bext](https://github.com/ublue-os/bext) as a plugin!
