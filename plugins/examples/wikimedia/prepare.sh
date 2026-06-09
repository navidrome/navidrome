#!/bin/bash
set -eou pipefail

# Function to check if a command exists
command_exists () {
  command -v "$1" >/dev/null 2>&1
}

# Function to compare version numbers for "less than"
version_lt() {
  test "$(echo "$@" | tr " " "\n" | sort -V | head -n 1)" = "$1" && test "$1" != "$2"
}

missing_deps=0

# Check for Go
if ! (command_exists go); then
  missing_deps=1
  echo "❌ Go (supported version between 1.20 - 1.24) is not installed."
  echo ""
  echo "To install Go, visit the official download page:"
  echo "👉 https://go.dev/dl/"
  echo ""
  echo "Or install it using a package manager:"
  echo ""
  echo "🔹 macOS (Homebrew):"
  echo "    brew install go"
  echo ""
  echo "🔹 Ubuntu/Debian:"
  echo "    sudo apt-get -y install golang-go"
  echo ""
  echo "🔹 Arch Linux:"
  echo "    sudo pacman -S go"
  echo ""
  echo "🔹 Windows:"
  echo "    scoop install go"
  echo ""
fi

# Check for the right version of Go, needed by TinyGo (supports go 1.20 - 1.24)
if (command_exists go); then
  compat=0
  for v in `seq 20 24`; do
    if (go version | grep -q "go1.$v"); then
      compat=1
    fi
  done

  if [ $compat -eq 0 ]; then
    echo "❌ Supported Go version is not installed. Must be Go 1.20 - 1.24."
    echo ""
  fi
fi

ARCH=$(arch)

# Check for TinyGo and its version
if ! (command_exists tinygo); then
  missing_deps=1
  echo "❌ TinyGo is not installed."
  echo ""
  echo "To install TinyGo, visit the official download page:"
  echo "👉 https://tinygo.org/getting-started/install/"
  echo ""
  echo "Or install it using a package manager:"
  echo ""
  echo "🔹 macOS (Homebrew):"
  echo "    brew tap tinygo-org/tools"
  echo "    brew install tinygo"
  echo ""
  echo "🔹 Ubuntu/Debian:"
  echo "    wget https://github.com/tinygo-org/tinygo/releases/download/v0.34.0/tinygo_0.34.0_$ARCH.deb"
  echo "    sudo dpkg -i tinygo_0.34.0_$ARCH.deb"
  echo ""
  echo "🔹 Arch Linux:"
  echo "    pacman -S extra/tinygo"
  echo ""
  echo "🔹 Windows:"
  echo "    scoop install tinygo"
  echo ""
else
  # Check TinyGo version
  tinygo_version=$(tinygo version | grep -o '[0-9]\+\.[0-9]\+\.[0-9]\+' | head -n1)
  if version_lt "$tinygo_version" "0.34.0"; then
    missing_deps=1
    echo "❌ TinyGo version must be >= 0.34.0 (current version: $tinygo_version)"
    echo "Please update TinyGo to a newer version."
    echo ""
  fi
fi

go install golang.org/x/tools/cmd/goimports@latest
