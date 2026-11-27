package ass

import (
	"io"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/psyb0t/ctxerrors"
)

const (
	sectionInfo   = "[script info]"
	sectionStyles = "[v4+ styles]"
	sectionEvents = "[events]"
	maxInputBytes = 64 * 1024 * 1024
	bom           = "\ufeff"
)

type record struct {
	line   int
	prefix string
	ending string
	names  []string
	values []string
}

// Document keeps the original ASS text and locates the fields that can be edited.
type Document struct {
	lines     []string
	info      map[string]record
	infoStart int
	styles    []record
	events    []record
}

// Contents contains the parsed fields. All returned maps are independent of the document.
type Contents struct {
	Info   map[string]string
	Styles []map[string]string
	Events []map[string]string
}

// Parse reads a UTF-8 ASS v4+ file, preserving its original bytes.
func Parse(reader io.Reader) (*Document, error) {
	data, err := io.ReadAll(io.LimitReader(reader, maxInputBytes+1))
	if err != nil {
		return nil, ctxerrors.Wrap(err, "read ASS document")
	}
	if len(data) > maxInputBytes {
		return nil, ctxerrors.Wrap(ErrInputTooLarge, "read ASS document")
	}
	if !utf8.Valid(data) || strings.ContainsRune(string(data), 0) {
		return nil, ctxerrors.Wrap(ErrInvalidDocument, "ASS must be UTF-8 without NUL bytes")
	}

	document := &Document{
		lines:     slices.Collect(strings.SplitAfterSeq(string(data), "\n")),
		info:      make(map[string]record),
		infoStart: -1,
	}
	if err := document.index(); err != nil {
		return nil, ctxerrors.Wrap(err, "parse ASS document")
	}

	return document, nil
}

func (d *Document) index() error {
	var section string
	var fields []string
	for index, raw := range d.lines {
		content := strings.TrimSuffix(strings.TrimSuffix(raw, "\n"), "\r")
		ending := raw[len(content):]
		line := strings.TrimSpace(strings.TrimPrefix(content, bom))

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(line)
			fields = nil
			if section == sectionInfo {
				d.infoStart = index
			}
			continue
		}

		key, payload, hasColon := strings.Cut(content, ":")
		if !hasColon || strings.HasPrefix(line, ";") {
			continue
		}
		kind := strings.ToLower(strings.TrimSpace(key))
		value := strings.TrimPrefix(payload, " ")
		prefix := content[:len(content)-len(value)]

		switch section {
		case sectionInfo:
			d.info[kind] = record{index, prefix, ending, []string{strings.TrimSpace(key)}, []string{value}}
		case sectionStyles, sectionEvents:
			if kind == "format" {
				fields = strings.Split(value, ",")
				for position := range fields {
					fields[position] = strings.TrimSpace(fields[position])
				}
				continue
			}
			if err := d.indexRow(index, prefix, ending, kind, section, value, fields); err != nil {
				return ctxerrors.Wrapf(err, "line %d", index+1)
			}
		}
	}
	if d.infoStart < 0 {
		return ctxerrors.Wrap(ErrInvalidDocument, "missing [Script Info]")
	}

	return nil
}

func (d *Document) indexRow(index int, prefix, ending, kind, section, value string, fields []string) error {
	isStyle := section == sectionStyles && kind == "style"
	isEvent := section == sectionEvents && (kind == "dialogue" || kind == "comment")
	if !isStyle && !isEvent {
		return nil
	}

	if len(fields) == 0 || (isEvent && !strings.EqualFold(fields[len(fields)-1], "Text")) {
		return ctxerrors.Wrap(ErrInvalidDocument, "expected a Format line with Text last for events")
	}
	values := strings.SplitN(value, ",", len(fields))
	if len(values) != len(fields) || (isStyle && strings.Contains(values[len(values)-1], ",")) {
		return ctxerrors.Wrap(ErrInvalidDocument, "record does not match its Format line")
	}

	item := record{index, prefix, ending, fields, values}
	if isStyle {
		d.styles = append(d.styles, item)
	} else {
		d.events = append(d.events, item)
	}

	return nil
}

func (r record) fieldIndex(name string) int {
	return slices.IndexFunc(r.names, func(field string) bool {
		return strings.EqualFold(field, name)
	})
}

func (r record) field(name string) string {
	index := r.fieldIndex(name)
	if index >= 0 {
		return r.values[index]
	}
	return ""
}

func (r record) fields() map[string]string {
	result := make(map[string]string, len(r.names))
	for index, name := range r.names {
		result[name] = r.values[index]
	}
	return result
}

// Contents returns the script settings, styles, and events in file order.
func (d *Document) Contents() Contents {
	result := Contents{
		Info:   make(map[string]string, len(d.info)),
		Styles: make([]map[string]string, 0, len(d.styles)),
		Events: make([]map[string]string, 0, len(d.events)),
	}
	for _, item := range d.info {
		maps.Copy(result.Info, item.fields())
	}
	for _, style := range d.styles {
		result.Styles = append(result.Styles, style.fields())
	}
	for _, event := range d.events {
		result.Events = append(result.Events, event.fields())
	}

	return result
}

func (d *Document) String() string { return strings.Join(d.lines, "") }

// Validate checks style names, references, and ASS dialogue timestamps.
func (d *Document) Validate() error {
	names := make(map[string]bool, len(d.styles))
	for _, style := range d.styles {
		name := strings.TrimSpace(style.field("Name"))
		if name == "" || names[name] {
			return ctxerrors.Wrapf(ErrInvalidDocument, "line %d has an empty or duplicate style", style.line+1)
		}
		names[name] = true
	}
	for _, event := range d.events {
		name := strings.TrimSpace(event.field("Style"))
		if !names[name] {
			return ctxerrors.Wrapf(ErrInvalidDocument, "line %d refers to missing style %q", event.line+1, name)
		}
		start, startOK := timestamp(event.field("Start"))
		end, endOK := timestamp(event.field("End"))
		if !startOK || !endOK || end <= start {
			return ctxerrors.Wrapf(ErrInvalidDocument, "line %d has invalid event timing", event.line+1)
		}
	}

	return nil
}

func timestamp(value string) (time.Duration, bool) {
	clock, decimal, ok := strings.Cut(strings.TrimSpace(value), ".")
	pieces := strings.Split(clock, ":")
	if !ok || len(pieces) != 3 || len(decimal) != 2 || len(pieces[1]) != 2 || len(pieces[2]) != 2 {
		return 0, false
	}
	limits := []int{99999, 59, 59, 99}
	units := []time.Duration{time.Hour, time.Minute, time.Second, 10 * time.Millisecond}
	var result time.Duration
	for index, part := range append(pieces, decimal) {
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 || number > limits[index] {
			return 0, false
		}
		result += time.Duration(number) * units[index]
	}

	return result, true
}
