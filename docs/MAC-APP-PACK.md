# Mac app pack: fork Ollama app, Plugins visible, tool call in chat
Status: READY TO RUN ON MAC. Nothing in this pack is claimed as done until screenshots 1, 2, and 3 arrive back here.
Stage 1: stage1-fork-app-2026-11-04-01. Stage 2: stage2-fork-app-2026-11-04-01.
Order matters: app first, demo page last. Do not start with the demo page, and do not open the installed Ollama app from Applications.

## What this proves, in plain words
That in the real fork Ollama app on a Mac, with a tool-capable model, an enabled plugin tool is actually called by the model in chat, the result comes back, and the model answers using that result.

## Step 0: Checkout, stop if anything is missing
- Fork repo, branch plugin-seed, commit c8d2fa5a31c02f559da197b98bd8106014ad4d86 or later. This pack itself is in the fork at docs/MAC-APP-PACK.md, so pulling plugin-seed gets the pack too.
- Verify these files exist: app/tools/plugin.go, app/tools/plugin_test.go, app/ui/app/src/components/Plugins.tsx.
- If any is missing, stop and report the exact missing file. Do not substitute the stock app, and do not use the companion seed repo as a substitute for the fork.

## Step 1: Fork server, prove it is the fork, with a plugin folder set
- Stop the stock Ollama server if it is running on 11434, and keep the installed Ollama app stopped, so it cannot restart a stock server on 11434.
- Build a fork binary named ollama in the fork repo root, from plugin-seed. The desktop app helper finds ollama in PATH, so the binary name must be ollama, not ollama-plugins.
- Set OLLAMA_PLUGINS_DIR to the seed plugins folder, the folder that contains browser, computer, and skill-creator, for example /path/to/ollama-plugin-seed/plugins, using the real Mac path, not this example path. Do not point it at the fork repo plugins folder, that folder holds Go code, not plugin folders.
- Build and run the fork server from plugin-seed on 11434, with OLLAMA_PLUGINS_DIR set as above.
- Check: GET http://127.0.0.1:11434/api/plugins must return browser, computer, and skill-creator, not an empty list.
- Why: stock Ollama does not have /api/plugins, so this check proves the server is the fork. If it returns 404, stop, the server is stock, not the fork. If it returns an empty list, stop, OLLAMA_PLUGINS_DIR is wrong or missing, do not go on to the Plugins page.

## Step 2: Fork desktop app, production UI build, not dev mode, fork first in PATH
- In app/ui/app: npm install, then npm run build, with no new dependencies.
- Start the source app with the fork repo root first in PATH, for example PATH=/path/to/ollama:$PATH go run ./cmd/app from the app folder, using the real Mac path, so the helper resolves the fork ollama binary built in step 1, not /opt/homebrew/bin/ollama.
- In app: go generate ./..., then go run ./cmd/app with that PATH set.
- Do not open the installed Ollama app from Applications.
- Do not use the -dev flag in this run, because -dev fixes the API port to 3001.
- After the app starts, re-check GET http://127.0.0.1:11434/api/plugins. If it is now 404, stop, the app launched stock Ollama again, and the PATH fix did not take effect.
- If the build fails, report the exact error, do not work around it with the stock app.

## Step 3: App check, screenshot 1
- Open the fork app built in step 2.
- Screenshot 1 must show: sidebar with Plugins between Chat and Settings, and the chat composer with the puzzle button.
- If Plugins is missing, stop. That is a gap, not a pass, and it must be reported, not hidden by opening the stock app.

## Step 4: Plugins check, screenshot 2
- Open Plugins in the fork app.
- Screenshot 2 must show: Browser with the Chrome icon, Computer with the gradient icon, and Browser enabled.
- Record the enabled state in plain words with the screenshot.

## Step 5: Model check, before any chat proof
- Record the model name used for chat.
- Confirm it is tool-capable. Use a chat with no attachments.
- If the model cannot call tools, pick another tool-capable model and record it. A chat with a non-tool model does not count.

## Step 6: Seed server
- From the seed repo on main, commit 478c68d8, run the seed server on 127.0.0.1:11435.
- Check: GET http://127.0.0.1:11435/api/tools lists browser tools.
- If the seed copy has no webmcp-demo/ folder, pull seed main first, do not push anything.

## Step 7: Demo, only after steps 1 to 6 pass
- Check which port is free, record one port, and use that exact port everywhere in this step.
- Serve webmcp-demo on that recorded port, open that exact origin in Chrome.
- In the Ollama Chrome extension popup, allow only that exact origin.
- On the demo page, click List registered tools, record local_echo and local_time.
- Note: the demo page text saying 3000 does not override the recorded port. If the recorded port is 3001, the allowed origin is http://localhost:3001, and the page opened is http://localhost:3001/.
- Do not allow any other site, and do not open Chrome tabs for other work in this step.

## Step 8: Chat proof, screenshot 3
- In the fork app, with the model from step 5, ask a question that should trigger a browser tool.
- Screenshot 3 must show, all in one uncropped screenshot: the question, the tool call name, the tool result, and the model answer using that result.
- Do not crop out the tool call or the result.

## Step 9: Send back here
Send screenshots 1, 2, and 3, plus, in plain words: fork commit, model name, fork server /api/plugins result, seed /api/tools result, recorded demo port and origin, and whether step 8 worked or what happened instead.
Until screenshots 1, 2, and 3 arrive, the finish check is NOT met.

## Rules for this pack
- Local only. No push, deploy, post, send, pay, delete, or publish.
- Computer actions stay dry-run. No real computer action, no real account site, no site other than the recorded local demo origin.
- Kill only servers started by this run when you finish, and report any server left running.
