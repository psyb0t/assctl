package ass

import (
	"io"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/psyb0t/ctxerrors"
)

const (
	maxInputBytes = 64 * 1024 * 1024
	bom           = "\ufeff"
	commentPrefix = ";"
	timeFields    = 3
	clockDigits   = 2
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
			section = line
			fields = nil
			if strings.EqualFold(section, headerScriptInfo) {
				d.infoStart = index
			}
			continue
		}

		key, payload, hasColon := strings.Cut(content, ":")
		if !hasColon || strings.HasPrefix(line, commentPrefix) {
			continue
		}
		kind := strings.TrimSpace(key)
		value := strings.TrimPrefix(payload, " ")
		prefix := content[:len(content)-len(value)]

		switch {
		case strings.EqualFold(section, headerScriptInfo):
			d.info[strings.ToLower(kind)] = record{index, prefix, ending, []string{kind}, []string{value}}
		case strings.EqualFold(section, headerStyles), strings.EqualFold(section, headerEvents):
			if strings.EqualFold(kind, recordFormat) {
				fields = strings.Split(value, fieldSeparator)
				seen := make(map[string]bool, len(fields))
				for position := range fields {
					fields[position] = strings.TrimSpace(fields[position])
					name := strings.ToLower(fields[position])
					if name == "" || seen[name] {
						return ctxerrors.Wrapf(ErrInvalidDocument, "line %d has empty or duplicate Format columns", index+1)
					}
					seen[name] = true
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
	isStyle := strings.EqualFold(section, headerStyles) && strings.EqualFold(kind, recordStyle)
	isEvent := strings.EqualFold(section, headerEvents) &&
		(strings.EqualFold(kind, recordDialogue) || strings.EqualFold(kind, recordComment))
	if !isStyle && !isEvent {
		return nil
	}

	if len(fields) == 0 || (isEvent && !strings.EqualFold(fields[len(fields)-1], columnText)) {
		return ctxerrors.Wrap(ErrInvalidDocument, "expected a Format line with Text last for events")
	}
	values := strings.SplitN(value, fieldSeparator, len(fields))
	if len(values) != len(fields) || (isStyle && strings.Contains(values[len(values)-1], fieldSeparator)) {
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

// String returns the document text, including every untouched byte of the original.
func (d *Document) String() string { return strings.Join(d.lines, "") }

// Validate checks style names, references, and ASS dialogue timestamps.
func (d *Document) Validate() error {
	names := make(map[string]bool, len(d.styles))
	for _, style := range d.styles {
		name := strings.TrimSpace(style.field(columnName))
		if name == "" || names[name] {
			return ctxerrors.Wrapf(ErrInvalidDocument, "line %d has an empty or duplicate style", style.line+1)
		}
		names[name] = true
	}
	for _, event := range d.events {
		name := strings.TrimSpace(event.field(columnStyle))
		if !names[name] {
			return ctxerrors.Wrapf(ErrInvalidDocument, "line %d refers to missing style %q", event.line+1, name)
		}
		start, startOK := timestamp(event.field(columnStart))
		end, endOK := timestamp(event.field(columnEnd))
		if !startOK || !endOK || end <= start {
			return ctxerrors.Wrapf(ErrInvalidDocument, "line %d has invalid event timing", event.line+1)
		}
	}

	return nil
}

func timestamp(value string) (time.Duration, bool) {
	clock, decimal, ok := strings.Cut(strings.TrimSpace(value), ".")
	pieces := strings.Split(clock, ":")
	if !ok || len(pieces) != timeFields || len(decimal) != clockDigits ||
		len(pieces[1]) != clockDigits || len(pieces[2]) != clockDigits {
		return 0, false
	}
	limits := []int{int(math.MaxInt64 / int64(time.Hour)), maxClockMinute, maxClockSecond, maxClockCenti}
	units := []time.Duration{time.Hour, time.Minute, time.Second, centisecond}
	var result time.Duration
	for index, part := range append(pieces, decimal) {
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 || number > limits[index] {
			return 0, false
		}
		partDuration := time.Duration(number) * units[index]
		if partDuration > time.Duration(math.MaxInt64)-result {
			return 0, false
		}
		result += partDuration
	}

	return result, true
}
