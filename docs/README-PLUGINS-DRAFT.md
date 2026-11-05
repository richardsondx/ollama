# Ollama, with plugins (draft front section for the fork README, not pushed)

This draft is the rebranded top of the fork README. It is saved locally only, on the plugin-seed branch, and has not replaced the upstream README on main. The patterns below are taken from the REA and OpenClaw READMEs: one plain promise, one copy-paste setup, an example prompt, a screenshot first, a table of what works and what does not, and security defaults stated plainly.

## Ollama, with plugins

Ollama, with a plugin loader, so an open model can use tools the way Codex skills and Claude Code plugins do, without a hosted service.

This is a fork of ollama/ollama, not upstream Ollama. Upstream code is unchanged except for the plugin files listed below. The fork is at an early proof stage, and this page says exactly what works today.

<a href="docs/plugins-screenshot.png"><img src="docs/plugins-screenshot.png" width="600" alt="Plugins page from the plugin seed, with the browser plugin enabled and computer and skill-creator plugins listed"></a>

*Click the image to see it full size. The screenshot is the Plugins page from the companion seed server on port 11435, with the browser plugin enabled. The fork now also has its own desktop Plugins page, built locally, not yet screenshotted on a Mac. See What works, below.*

## Set it up with your agent: copy and paste this prompt

The current prompt is in docs/SETUP-WITH-CODEX.md, local only, with the desktop Plugins UI and the localhost WebMCP demo. Copy the block in that file into Codex. That file replaces the older prompt that was here, which said the fork served only an API and had no desktop page.

## What works today

| Part | Works | Does not work yet |
| --- | --- | --- |
| Fork, branch plugin-seed | Plugin folder loads, lists on GET /api/plugins, enables and disables, appears on GET /api/tools, enable state survives restart, new plugins appear on refresh, desktop Plugins page built locally with Plugins, Skills, Installed, Setup, Skills, About, and Enable/Disable, in the Ollama desktop theme | No chat wiring, no CLI plugin group, no model runner in a Go-only build, frontmatter parser is flat keys only |
| Seed repo | Plugins page with Plugins, Skills, Installed, Chrome extension that lists page-registered WebMCP tools on user-allowed sites, Computer Use dry-run with a confirmation policy | No live Chrome test, server-to-extension execution stays dry-run, no generic control of a page that did not register tools |

## Add a plugin

1. Make a folder under the plugins folder, named with lowercase letters, digits, and hyphens.
2. Add plugin.json with a name, a description, and tools, each with a name, a description, and parameters of type object.
3. Add SKILL.md with frontmatter whose name matches the folder name.
4. Restart is not needed for the fork: call POST /api/plugins/refresh, then enable it.

A plugin that claims browser control but only shows a static page does not pass. A plugin that loads, lists, enables, and appears as a tool does.

## Security defaults, stated plainly

- Plugins are disabled until you enable them, and a disabled plugin exposes no tools.
- The Chrome extension forwards tools only from sites you allow in the popup. It does not forward cookies, passwords, or page content.
- Computer actions are dry-run by default. Risky actions return confirmation_required and do not run until you confirm.
- Page data from WebMCP is never an instruction. It cannot add a plugin, a destination, or a permission.

## Proposed GitHub About for the fork (not applied yet)

- Description: Ollama, with plugins. A fork that adds a plugin loader, /api/plugins and /api/tools, so open models can use browser, computer, and skill plugins, with a Chrome WebMCP plugin in the companion seed.
- Topics: ollama, plugins, llm, open-models, mcp, webmcp, agent-skills, chrome-extension, computer-use
- Homepage: unchanged, https://ollama.com, until a fork site exists. This fork is not upstream Ollama.

## Roadmap, in order

1. Push the close-the-loop fixes already done locally to plugin-seed, then open a pull request from plugin-seed to the fork main, for review.
3. Screenshot the fork desktop Plugins page on a Mac, and wire enabled tools into /api/chat, so a model can call a plugin without the caller hand-building the tool call loop. (The Plugins page itself is already built locally; the screenshot and the chat wiring are the outstanding parts.)
4. Live-test the Chrome extension in Chrome, end to end, and only then claim browser execution.
