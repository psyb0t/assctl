#!/usr/bin/env bash
# Checks a built assctl image offline: the stamped version; for every built-in
# preset, captions with --embed-preset-fonts and an FFmpeg render in which
# libass picks the preset's installed font; and assctl burn on a test video.
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
ffmpeg -v error -f lavfi -i color=c=black:s=540x960:r=24:d=1 -f lavfi -i sine=d=1 \
	-c:v libx264 -pix_fmt yuv420p -c:a aac clip.mp4
assctl burn clip.mp4 --subtitles tiktok-bold.ass --output burned.mp4 2>/dev/null
before="$(ffprobe -v error -select_streams v:0 -show_entries stream=codec_name,width,height,pix_fmt -of csv=p=0 clip.mp4)"
after="$(ffprobe -v error -select_streams v:0 -show_entries stream=codec_name,width,height,pix_fmt -of csv=p=0 burned.mp4)"
[ "$before" = "$after" ] || { echo "burn changed the video format: $before -> $after" >&2; exit 1; }
[ "$(ffprobe -v error -show_entries stream=codec_type -of csv=p=0 burned.mp4 | tr "\n" " ")" = "video audio " ] ||
	{ echo "burn did not keep the audio stream" >&2; exit 1; }
echo "burn: ok ($after)"
'
