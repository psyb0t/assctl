# Setup

Install with Go 1.26 or newer.

```bash
go install github.com/psyb0t/assctl/cmd/assctl@latest
assctl version
```

## Named looks

- `black-gold` uses a black background, white text, and gold as its secondary karaoke color.
- `ivory-navy` uses a navy background, ivory text, and sky blue as its secondary karaoke color.
- `mint-plum` uses a plum background, mint text, and peach as its secondary karaoke color.
- `clean` uses an outline on a clear background.

Run `assctl presets --json` for complete parameters. Existing inline tags remain intact and take precedence over the style definition.

## Precise edits

Inspect a document first. Its JSON output shows the exact field names. Supply an edit spec with `Info`, `Styles`, or `Events`. An event index starts at one. Use typed `Override` fields to add tags without replacing dialogue text.

```json
{"Styles":[{"Name":"Default","Fields":{"Fontsize":"52"}}]}
```

```bash
assctl edit --spec edits.json --output updated.ass captions.ass
```

See the [CLI guide](https://github.com/psyb0t/assctl/blob/main/docs/cli.md) for the full creation and editing examples.
