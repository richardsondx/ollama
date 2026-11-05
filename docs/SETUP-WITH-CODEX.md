# Set it up with Codex: copy and paste this prompt

Copy everything in the block below into Codex, and let Codex set it up on your machine. Local access only. Change only the model name and the port, if you need to.

```
Set up Ollama with plugins on my machine, locally only, end to end, and prove each step works before moving on.

Local only means: no push, no deploy, no publish, no post, no send, no pay, and no delete. Do not change Ollama model runners, and do not copy OpenAI code, the ChatGPT extension, or any proprietary computer control package.

There are two parts, and they are not the same repo:
1. This fork, branch plugin-seed: the Go plugin loader inside Ollama itself, plus the Plugins page inside the Ollama desktop app.
2. The companion seed, https://github.com/richardsondx/ollama-plugin-seed : the standalone Plugins page on port 11435, the Chrome extension, and a local WebMCP demo page.

Fork part, in order:
1. Get a local copy of this fork, branch plugin-seed. If a local copy already exists on branch plugin-seed and contains app/ui/app/src/components/Plugins.tsx, use that copy. If not, clone this branch now, you are explicitly allowed to clone it for this setup:
   git clone --branch plugin-seed https://github.com/richardsondx/ollama.git
   Do not use main. Main is still upstream Ollama with no plugin code. After cloning, verify app/ui/app/src/components/Plugins.tsx exists. If it does not, stop and tell me, do not assume it is there. Do not use the companion seed repo as a substitute for the fork, and do not use a Downloads folder that is not a git repository.
2. Check Go 1.26.10 is available, matching the fork go.mod. If it is not, install it and say so, do not build with a different Go version and claim it matches.
3. Build the desktop app UI: in app/ui/app, run npm install, then npm run build. It must pass TypeScript and Vite with no errors. If the route file needs regenerating, run vite build once first, then run the full build again.
4. Build the fork server: go build -o ollama-plugins .
5. Make a plugins folder with one test plugin folder inside it, containing plugin.json and SKILL.md, with the frontmatter name matching the folder name.
6. Run the fork server on a port that does not clash with an existing Ollama, for example OLLAMA_HOST=127.0.0.1:11436, with OLLAMA_PLUGINS_DIR pointing at your plugins folder and OLLAMA_PLUGIN_STATE pointing at a state file you name.
7. Prove the API live: GET /api/plugins lists your plugin, POST /api/plugins/<name>/enable returns enabled true, GET /api/tools returns its tools in Ollama chat tool shape, and restarting the server keeps the plugin enabled, because the state file is read back.
8. Add a second plugin folder after start, then call POST /api/plugins/refresh and show it appears without a restart.
9. Prove the desktop Plugins UI: open the Ollama desktop app built from this fork, open Plugins in the left nav, and also open it from the puzzle icon in the chat input. Show that the page lists your plugin, with Plugins, Skills, and Installed tabs, a Setup tool list, a Skills list, and an About section, in the Ollama desktop theme, white and neutral greys with black pill buttons. Enable and disable must work from that page.

Seed part, in order, local only:
10. Clone https://github.com/richardsondx/ollama-plugin-seed, run its tests, then run python3 scripts/run_server.py. It serves the Plugins page on http://127.0.0.1:11435, with Plugins, Skills, and Installed tabs.
11. Set up the Chrome extension, with my help for the Chrome steps only: tell me, in exact steps, how to load the seed chrome-extension/ folder unpacked in Chrome, at chrome://extensions, with Developer mode on. Wait for me to confirm it is loaded. Do not claim it is loaded before I confirm.
12. Local WebMCP test only, no external site, no example.com: serve the seed webmcp-demo/ folder locally with python3 -m http.server 3000, open http://localhost:3000/ in Chrome, with the WebMCP testing flag on, and tell me how to allow http://localhost:3000 in the extension popup. Only forward tools from sites I allow. Never read or share cookies, passwords, local storage, or page content. The extension only lists tools a page itself registered with WebMCP. It does not give generic control of a page that did not register tools. The demo registers local_echo and local_time, and those are the only tools to expect on localhost.
13. Connect it to a chat call: take the tool definitions from /api/tools, pass them in an Ollama /api/chat request, execute any returned tool calls through the plugin server, and return the results as tool messages.

Rules for the whole setup:
- Computer actions stay dry-run, and any action that deletes, sends, pays, changes permissions, or installs anything needs my confirmation right before it runs, not in advance.
- If a step fails, stop at that step, show the exact error, and tell me what would unblock it. Do not skip the step and claim success.
- At the end, give me a short checklist: what works, what is still manual, and the exact URLs: the desktop Plugins page in the Ollamapaca app, the seed Plugins page at http://127.0.0.1:11435, the fork server on the port you chose, the local demo at http://localhost:3000, and Ollama on http://127.0.0.1:11434 if it is also running.

What is honest, and what is not yet done:
- Enabling a plugin lists its tools at /api/tools. Chat does not call those tools by itself yet, so enabling alone does not change what the model can do in the desktop app. Do not claim otherwise.
- The Chrome extension has not been live-tested end to end in Chrome here. The localhost demo is the first live test.
```

If your agent asks which model to use, pick a tool-capable model you have installed, such as a Llama 3.1, Qwen 2.5 or 3, or Mistral class model.

Note for sharing: this prompt lives in this file, on branch plugin-seed in https://github.com/richardsondx/ollama. If Codex runs on a different machine with no local copy, step 1 above tells it to clone that branch, so it can get the Plugins UI and this prompt from GitHub.
