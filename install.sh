#!/bin/sh
# aios installer.
#
# This is the "there is no download" problem's answer: one
# command, and the operator has a working `aios`.
#
#   curl -fsSL https://raw.githubusercontent.com/Hamidullah02/aios/main/install.sh | sh
#
# What it does:
#
#   1. Picks a download for the operator's OS and CPU.
#   2. Downloads it from the latest GitHub release.
#   3. Checks it runs.
#   4. Puts it somewhere on the PATH.
#
# What it does NOT do: install Go, set up a workspace, or ask
# for an API key. The first run needs a key, and `aios init`
# tells the operator where to put it.
#
# Environment variables:
#
#   AIOS_VERSION   the release tag to install (default: latest)
#   AIOS_INSTALL   the directory to install into (default: /usr/local/bin
#                  when writable, else ~/.local/bin)
#   AIOS_REPO      the GitHub repo (default: Hamidullah02/aios)
#
# The repo is the GitHub one the releases live in. The Go module
# lives in its aios-system/ subdirectory, so the module path
# carries both: github.com/Hamidullah02/aios/aios-system. That is
# the path `go install` needs, and it is why the module path and
# the repo name are not the same string.

set -eu

REPO="${AIOS_REPO:-Hamidullah02/aios}"
VERSION="${AIOS_VERSION:-latest}"

say() { printf '%s\n' "$*" >&2; }
die() { printf 'aios: %s\n' "$*" >&2; exit 1; }

# --- pick the platform -------------------------------------------------

os="$(uname -s)"
arch="$(uname -m)"

case "$os" in
  Linux)  os="linux" ;;
  Darwin) os="darwin" ;;
  *)      die "unsupported OS: $os. Build from source instead: go install github.com/Hamidullah02/aios/aios-system/cmd/aios@latest" ;;
esac

case "$arch" in
  x86_64|amd64)  arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *)             die "unsupported CPU: $arch. Build from source instead: go install github.com/Hamidullah02/aios/aios-system/cmd/aios@latest" ;;
esac

asset="aios_${os}_${arch}.tar.gz"

# --- resolve the version ------------------------------------------------

if [ "$VERSION" = "latest" ]; then
  api="https://api.github.com/repos/$REPO/releases/latest"
  # A redirect is the cheapest way to ask GitHub for the latest
  # tag without parsing JSON.
  version_url="$(curl -fsSL -o /dev/null -w '%{url_effective}' "$api" 2>/dev/null || true)"
  VERSION="${version_url##*/tag/}"
  [ -n "$VERSION" ] || VERSION="latest"
fi

if [ "$VERSION" = "latest" ]; then
  url="https://github.com/$REPO/releases/latest/download/$asset"
else
  url="https://github.com/$REPO/releases/download/$VERSION/$asset"
fi

say "aios: downloading $asset ($VERSION)"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

if ! curl -fsSL "$url" -o "$tmp/$asset"; then
  die "could not download $url
The release may not have a build for this platform yet.
Build from source instead: go install github.com/Hamidullah02/aios/aios-system/cmd/aios@latest"
fi

# --- unpack -------------------------------------------------------------

tar -xzf "$tmp/$asset" -C "$tmp"
[ -f "$tmp/aios" ] || die "the download did not contain an 'aios' binary"

chmod +x "$tmp/aios"

# --- install ------------------------------------------------------------

if [ -n "${AIOS_INSTALL:-}" ]; then
  dest="$AIOS_INSTALL"
elif [ -w "/usr/local/bin" ]; then
  dest="/usr/local/bin"
else
  dest="$HOME/.local/bin"
  mkdir -p "$dest"
fi

if [ -w "$dest" ]; then
  mv "$tmp/aios" "$dest/aios"
else
  say "aios: $dest is not writable; installing into \$HOME/.local/bin instead"
  dest="$HOME/.local/bin"
  mkdir -p "$dest"
  mv "$tmp/aios" "$dest/aios"
fi

# --- check --------------------------------------------------------------

if ! "$dest/aios" --help >/dev/null 2>&1; then
  die "the installed binary did not run. Try: $dest/aios --help"
fi

# Report what was actually installed. An operator who needs to
# file a bug report needs the version of the binary they are
# looking at, not the version they thought they were running.
installed_version="$("$dest/aios" --version 2>/dev/null || echo 'unknown')"

say ""
say "aios is installed at $dest/aios"
say "  $installed_version"
case ":$PATH:" in
  *":$dest:"*) ;;
  *) say "NOTE: $dest is not on your PATH. Add it: export PATH=\"$dest:\$PATH\"" ;;
esac
say ""
say "Next:"
say "  1. Put your model key in a .env file next to where you will run aios:"
say "       echo 'OPENROUTER_API_KEY=sk-or-...' > .env"
say "  2. Start it:"
say "       aios web"
say ""
say "The UI opens in your browser. Type what you want done."
