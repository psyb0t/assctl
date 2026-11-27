---
title: Agent integrations
description: Use the assctl skill with Claude Code, Codex, and OpenClaw.
---

# Agent integrations

Install the [CLI](install.md) first. The [assctl skill](https://github.com/psyb0t/assctl/tree/main/.agents/skills/assctl) teaches agents to inspect subtitle files, apply presets, edit selected fields, and validate output. The executable must be on the agent's `PATH`, or the agent can run the [Docker image](install.md#docker) the same way.

## Claude Code and Codex

```bash
claude plugin marketplace add psyb0t/agents
claude plugin install assctl@psyb0t
```

```bash
codex plugin marketplace add psyb0t/agents
codex plugin add assctl@psyb0t
```

These commands require the assctl entry to be published in the [shared catalog](https://github.com/psyb0t/agents). Agents that read skill directories can load the repository's skill directly.

## OpenClaw

Tagged releases publish the skill to ClawHub.

```bash
openclaw skills install @psyb0t/assctl
```

## Example task

Ask the agent to inspect your ASS file, apply `ivory-navy`, save a new file, and validate it. Use a separate output while trying presets. A video renderer can then burn the subtitles into your video.
