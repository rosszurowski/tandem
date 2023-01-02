#!/bin/sh

set -eu

# Use a main function so that a partial download doesn't execute half a script.
main() {
  repository="rosszurowski/tandem"
  releases_url="https://github.com/${repository}/releases"
  bin_name="tandem"
  destination="$(pwd)"

  # Step 1: parse incoming args to determine the destination to install to.
  for i in "$@"; do
    case $i in
      -d=*|--dest=*)
        destination="${i#*=}"
        shift # past argument=value
      ;;
      *)
        # unknown option
      ;;
    esac
  done

  # Step 2: get the lowercased OS type, either `darwin` or `linux`.
  os=$(uname -s | awk '{print tolower($0)}')
  case ${os} in
    "darwin" | "linux" )
      # do nothing
    ;;
    *)
      echo ""
      echo "Error: unsupported OS '${os}'"
      echo "You can try manually downloading binaries here:"
      echo "${releases_url}"
      echo ""
      exit 1
    ;;
  esac

  # Step 3: get the OS architecture.
  arch=$(uname -m)
  case ${arch} in
    "x86_64" | "amd64" )
      arch="amd64"
    ;;
    "i386" | "i486" | "i586")
      arch="386"
    ;;
    "aarch64" | "arm64" | "arm")
      arch="arm64"
    ;;
    *)
      echo ""
      echo "Error: unsupported architecture '${arch}'"
      echo "You can try manually downloading binaries here:"
      echo "${releases_url}"
      echo ""
      exit 1
    ;;
  esac

  # Step 4: find the URL to download from the GitHub releases API.
  asset_uri=$(
    curl -H "Accept: application/vnd.github.v3+json" \
      -sSf "https://api.github.com/repos/${repository}/releases/latest" |
    grep '"browser_download_url"' |
    grep -o "http[^\"]*" |
    grep "_${os}_${arch}.tar.gz$" |
    head -n 1
  )

  if [ -z "$asset_uri" ]; then
    echo ""
    echo "Error fetching ${bin_name} download URL"
    echo "You can try downloading binaries here:"
    echo "${releases_url}"
    echo ""
    exit 1
  fi

  latest_version=$(
    echo "$asset_uri" |
    grep -o "download/[^_]*" |
    grep -o "[[:digit:]][^/]*" |
    head -n 1
  )
  tmp_dir="$(mktemp -d)"
  tmp_file="${tmp_dir}/${bin_name}.tmp.tar.gz"

  echo "Downloading ${bin_name} v${latest_version} from GitHub..."
  curl -fsSL "$asset_uri" -o "${tmp_file}"
  if [ -n "${destination}" ]; then
    mkdir -p "${destination}"
    tar -xz -f "${tmp_file}" -C "${destination}" "${bin_name}"
  else
    tar -xz -f "${tmp_file}" "${bin_name}"
  fi
  rm -f "${tmp_file}"

  echo "Installed ${bin_name} v${latest_version} to ${destination}"
}

main "$@"

