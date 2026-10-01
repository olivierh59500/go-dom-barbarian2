#!/bin/sh
set -eu

usage() {
    echo "Usage: $0 [--build-only]" >&2
    echo "Build the ARM64 Android application, then install and launch it unless --build-only is set." >&2
}

build_only=false
for option in "$@"; do
    case "$option" in
        --build-only) build_only=true ;;
        --help|-h) usage; exit 0 ;;
        *) usage; exit 2 ;;
    esac
done

project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
android_sdk=${ANDROID_HOME:-${ANDROID_SDK_ROOT:-}}
java_home_path=${JAVA_HOME:-}

if [ -z "$android_sdk" ] && [ -d /opt/homebrew/share/android-commandlinetools ]; then
    android_sdk=/opt/homebrew/share/android-commandlinetools
fi
if [ -z "$java_home_path" ] && [ -x /opt/homebrew/opt/openjdk@17/bin/java ]; then
    java_home_path=/opt/homebrew/opt/openjdk@17
fi
if [ -z "$java_home_path" ] && [ -x "/Applications/Android Studio.app/Contents/jbr/Contents/Home/bin/java" ]; then
    java_home_path="/Applications/Android Studio.app/Contents/jbr/Contents/Home"
fi

if [ -z "$android_sdk" ] || [ ! -f "$android_sdk/platforms/android-36/android.jar" ]; then
    echo "Android SDK 36 was not found. Set ANDROID_HOME or ANDROID_SDK_ROOT." >&2
    exit 1
fi
if [ -z "$java_home_path" ] || [ ! -x "$java_home_path/bin/java" ]; then
    echo "Java 17 was not found. Set JAVA_HOME." >&2
    exit 1
fi
if [ ! -x "$project_root/android/gradlew" ]; then
    echo "The Gradle wrapper is missing from android/." >&2
    exit 1
fi
if ! "$build_only" && [ ! -x "$android_sdk/platform-tools/adb" ]; then
    echo "ADB was not found. Install Android SDK platform-tools." >&2
    exit 1
fi

export ANDROID_HOME="$android_sdk"
export ANDROID_SDK_ROOT="$android_sdk"
export JAVA_HOME="$java_home_path"
export PATH="$java_home_path/bin:$android_sdk/platform-tools:$PATH"
export GOWORK=off

cd "$project_root"
mkdir -p android/app/libs
echo "Checking pinned Android dependencies..."
# Resolve the module graph before the binding tool generates its temporary module.
GOOS=android GOARCH=arm64 go list -m -tags=android all >/dev/null
echo "Building the Go/Ebitengine ARM64 library..."
go run github.com/hajimehoshi/ebiten/v2/cmd/ebitenmobile@v2.9.11 \
    bind \
    -target android/arm64 \
    -androidapi 23 \
    -javapkg com.olivierh.dombarbarian2 \
    -o android/app/libs/libdombarbarian2.aar \
    ./mobile

echo "Building the debug APK..."
"$project_root/android/gradlew" -p "$project_root/android" --console=plain :app:assembleDebug

apk_path="$project_root/android/app/build/outputs/apk/debug/app-debug.apk"
echo "APK: $apk_path"
if "$build_only"; then
    exit 0
fi

adb_path="$android_sdk/platform-tools/adb"
if [ -z "${ANDROID_SERIAL:-}" ]; then
    device_count=$("$adb_path" devices | awk 'NR > 1 && $2 == "device" { count++ } END { print count + 0 }')
    if [ "$device_count" -ne 1 ]; then
        echo "Exactly one authorized Android device is required; found $device_count. Set ANDROID_SERIAL to select one." >&2
        "$adb_path" devices -l >&2
        exit 1
    fi
    ANDROID_SERIAL=$("$adb_path" devices | awk 'NR > 1 && $2 == "device" { print $1; exit }')
    export ANDROID_SERIAL
fi
if [ "$("$adb_path" get-state)" != "device" ]; then
    echo "The selected Android device is not authorized or connected." >&2
    exit 1
fi

echo "Installing Dom Barbarian II..."
"$adb_path" install -r "$apk_path"
echo "Launching Dom Barbarian II..."
"$adb_path" shell am start -S -W -n com.olivierh.dombarbarian2/.MainActivity
