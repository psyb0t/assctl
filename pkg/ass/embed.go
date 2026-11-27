package ass

import (
	"slices"
	"strings"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxerrors/commerr"
)

// EmbedFonts returns a document with fonts added to its [Fonts] section. The
// section is created at the end of the file when it is missing. Every other
// line keeps its bytes. A font whose name is already embedded is rejected.
func (d *Document) EmbedFonts(fonts []EmbeddedFont) (*Document, error) {
	encoded, err := fontLines(fonts)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "encode embedded fonts")
	}
	if len(encoded) == 0 {
		return d, nil
	}

	lines := slices.Clone(d.lines)
	ending := d.lineEnding()
	start, end := d.fontsSection()
	if start < 0 {
		// SplitAfter leaves an empty final element when the file ends with a newline.
		if last := len(lines) - 1; last >= 0 && lines[last] == "" {
			lines = lines[:last]
		}
		if last := len(lines) - 1; last >= 0 && !strings.HasSuffix(lines[last], "\n") {
			lines[last] += ending
		}
		lines = append(lines, ending, headerFonts+ending)
		end = len(lines)
	} else {
		existing := d.embeddedFontNames(start, end)
		for _, font := range fonts {
			if existing[strings.ToLower(font.Name)] {
				return nil, ctxerrors.Wrapf(commerr.ErrInvalidValue, "font %q is already embedded", font.Name)
			}
		}
		if last := end - 1; !strings.HasSuffix(lines[last], "\n") {
			lines[last] += ending
		}
	}

	added := make([]string, 0, len(encoded))
	for _, line := range encoded {
		added = append(added, line+ending)
	}
	lines = slices.Insert(lines, end, added...)

	updated, err := Parse(strings.NewReader(strings.Join(lines, "")))
	if err != nil {
		return nil, ctxerrors.Wrap(err, "read document with embedded fonts")
	}

	return updated, nil
}

// fontsSection returns the line range of the [Fonts] section, from its header
// to the line before the next section, or -1 when there is none. A trailing
// blank line belongs to the section's end, so added fonts go before it.
func (d *Document) fontsSection() (int, int) {
	start := -1
	for index, raw := range d.lines {
		line := strings.TrimSpace(strings.TrimPrefix(raw, bom))
		isHeader := strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]")
		if !isHeader {
			continue
		}
		if start >= 0 {
			end := index
			for end > start+1 && strings.TrimSpace(d.lines[end-1]) == "" {
				end--
			}

			return start, end
		}
		if strings.EqualFold(line, headerFonts) {
			start = index
		}
	}
	if start < 0 {
		return -1, -1
	}
	end := len(d.lines)
	for end > start+1 && strings.TrimSpace(d.lines[end-1]) == "" {
		end--
	}

	return start, end
}

func (d *Document) embeddedFontNames(start, end int) map[string]bool {
	names := make(map[string]bool)
	for _, raw := range d.lines[start+1 : end] {
		name, ok := strings.CutPrefix(strings.TrimRight(raw, lineBreakChars), strings.TrimSpace(fontNamePrefix))
		if ok {
			names[strings.ToLower(strings.TrimSpace(name))] = true
		}
	}

	return names
}

func (d *Document) lineEnding() string {
	for _, line := range d.lines {
		if strings.HasSuffix(line, "\r\n") {
			return "\r\n"
		}
		if strings.HasSuffix(line, "\n") {
			return "\n"
		}
	}

	return "\n"
}
