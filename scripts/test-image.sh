#!/usr/bin/env bash
# Checks a built assctl image offline: the stamped version, and for every
# built-in preset, captions with --embed-preset-fonts and an FFmpeg render in
# which libass picks the preset's installed font.
set -euo pipefail

image="${1:?usage: test-image.sh IMAGE EXPECTED_VERSION}"
expected_version="${2:?usage: test-image.sh IMAGE EXPECTED_VERSION}"

run() {
	docker run --rm --network none --read-only --tmpfs /tmp --cap-drop ALL \
		--security-opt no-new-privileges:true "$@"
}

version="$(run "$image" version)"
if [ "$version" != "$expected_version" ]; then
	echo "image reports version $version, want $expected_version" >&2
	exit 1
fi

# The script runs inside the image, so its variables expand there.
# shellcheck disable=SC2016
run --entrypoint sh "$image" -ec '
cd /tmp
printf "%s" "{\"Width\":540,\"Height\":960,\"Dialogues\":[{\"Start\":0,\"End\":1000000000,\"Text\":\"CHECK\"}]}" > spec.json
for preset in $(assctl presets); do
	assctl captions --preset "$preset" --embed-preset-fonts --spec spec.json --output "$preset.ass"
	grep -q "^fontname: " "$preset.ass" || { echo "$preset: no embedded font" >&2; exit 1; }
	selected="$(ffmpeg -hide_banner -loglevel verbose -f lavfi -i color=c=black:s=540x960:d=1 \
		-vf "ass=$preset.ass" -frames:v 1 -f null - 2>&1 | grep "fontselect:" || true)"
	case "$selected" in
	*"-> /usr/local/share/fonts/assctl/"*) echo "$preset: ok" ;;
	*) echo "$preset: libass did not select an installed preset font: $selected" >&2; exit 1 ;;
	esac
done
'
