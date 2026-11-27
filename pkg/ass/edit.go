package ass

import (
	"maps"
	"slices"
	"strings"

	"github.com/psyb0t/ctxerrors"
)

// Edit contains changes to script settings, named styles, and numbered events.
type Edit struct {
	Info   map[string]string
	Styles []StyleEdit
	Events []EventEdit
}

// StyleEdit replaces selected Format columns in a named style.
type StyleEdit struct {
	Name   string
	Fields map[string]string
}

// EventEdit updates an event by its one-based index, including comment events.
type EventEdit struct {
	Index    int
	Fields   map[string]string
	Override *Override
}

// Apply returns a new document. The original remains available if an edit fails.
func (d *Document) Apply(edit Edit) (*Document, error) {
	lines := slices.Clone(d.lines)
	if err := d.editInfo(lines, edit.Info); err != nil {
		return nil, ctxerrors.Wrap(err, "edit script settings")
	}

	styleIndices := make(map[string]record, len(d.styles))
	changed := make(map[int]bool, len(edit.Styles)+len(edit.Events))
	for _, style := range d.styles {
		styleIndices[strings.TrimSpace(style.field("Name"))] = style
	}
	for _, change := range edit.Styles {
		style, ok := styleIndices[change.Name]
		if !ok {
			return nil, ctxerrors.Wrapf(ErrNotFound, "style %q", change.Name)
		}
		if changed[style.line] {
			return nil, ctxerrors.Wrapf(ErrInvalidValue, "style %q appears twice in the edit", change.Name)
		}
		line, err := style.replace(change.Fields, nil)
		if err != nil {
			return nil, ctxerrors.Wrapf(err, "edit style %q", change.Name)
		}
		lines[style.line] = line
		changed[style.line] = true
	}

	for _, change := range edit.Events {
		if change.Index < 1 || change.Index > len(d.events) {
			return nil, ctxerrors.Wrapf(ErrNotFound, "event %d", change.Index)
		}
		event := d.events[change.Index-1]
		if changed[event.line] {
			return nil, ctxerrors.Wrapf(ErrInvalidValue, "event %d appears twice in the edit", change.Index)
		}
		line, err := event.replace(change.Fields, change.Override)
		if err != nil {
			return nil, ctxerrors.Wrapf(err, "edit event %d", change.Index)
		}
		lines[event.line] = line
		changed[event.line] = true
	}

	updated, err := Parse(strings.NewReader(strings.Join(lines, "")))
	if err != nil {
		return nil, ctxerrors.Wrap(err, "read edited ASS document")
	}
	if err := updated.Validate(); err != nil {
		return nil, ctxerrors.Wrap(err, "validate edited ASS document")
	}

	return updated, nil
}

func (d *Document) editInfo(lines []string, edits map[string]string) error {
	header := lines[d.infoStart]
	ending := "\n"
	if strings.HasSuffix(header, "\r\n") {
		ending = "\r\n"
	}
	if !strings.HasSuffix(header, "\n") && len(edits) > 0 {
		header += ending
	}

	var additions []string
	for _, name := range slices.Sorted(maps.Keys(edits)) {
		value := edits[name]
		if strings.TrimSpace(name) != name || name == "" || strings.ContainsAny(name, "[]:;\r\n\x00") ||
			strings.ContainsAny(value, "\r\n\x00") {
			return ctxerrors.Wrapf(ErrInvalidValue, "invalid setting %q", name)
		}
		if item, ok := d.info[strings.ToLower(name)]; ok {
			lines[item.line] = item.prefix + value + item.ending
		} else {
			additions = append(additions, name+": "+value+ending)
		}
	}
	lines[d.infoStart] = header + strings.Join(additions, "")

	return nil
}

func (r record) replace(fields map[string]string, override *Override) (string, error) {
	values := slices.Clone(r.values)
	seen := make(map[int]bool, len(fields))
	for name, value := range fields {
		index := r.fieldIndex(name)
		if index < 0 {
			return "", ctxerrors.Wrapf(ErrNotFound, "field %q", name)
		}
		if seen[index] {
			return "", ctxerrors.Wrapf(ErrInvalidValue, "field %q appears twice in the edit", name)
		}
		if strings.ContainsAny(value, "\r\n\x00") || (!strings.EqualFold(name, "Text") && strings.ContainsRune(value, ',')) {
			return "", ctxerrors.Wrapf(ErrInvalidValue, "field %q contains a record delimiter", name)
		}
		values[index] = value
		seen[index] = true
	}
	if override != nil {
		index := r.fieldIndex("Text")
		if index < 0 {
			return "", ctxerrors.Wrap(ErrInvalidDocument, "event is missing its Text field")
		}
		tags, err := override.Tags()
		if err != nil {
			return "", ctxerrors.Wrap(err, "encode event overrides")
		}
		values[index] = tags + values[index]
	}

	return r.prefix + strings.Join(values, ",") + r.ending, nil
}
