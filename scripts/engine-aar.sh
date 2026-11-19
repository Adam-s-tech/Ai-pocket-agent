#!/usr/bin/env bash
# Packs the Android engine builds into an .aar for Google Play builds,
# which can't download native code: giode puts its lib/arm64-v8a
# libraries into the app, and the app built with -tags play loads the
# one for the phone's CPU (internal/local/builtin_play.go).
#
#   scripts/engine-aar.sh   # build/engine/libpocketllama-android-arm64-{armv8.2,armv8}.so
#                           # -> build/engine/pocketllama-play.aar
set -euo pipefail
cd "$(dirname "$0")/.."
python3 - <<'PY'
import zipfile
src = "build/engine/libpocketllama-android-arm64-{}.so"
names = {"armv8.2": "libpocketllama_armv82.so", "armv8": "libpocketllama_armv8.so"}
out = "build/engine/pocketllama-play.aar"
with zipfile.ZipFile(out, "w", zipfile.ZIP_DEFLATED) as z:
    for variant, name in names.items():
        z.write(src.format(variant), "jni/arm64-v8a/" + name)
print(out)
PY
