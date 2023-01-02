#!/usr/bin/env sh
# Writes to src/index.ts after a delay. Used to simulate an update
# to the esbuild/tailwindcss watchers in the demo.

sleep "$1"
echo "$2" > src/index.ts
