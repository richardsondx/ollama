# Mac app pack: fork Ollama app, Plugins visible, tool call in chat
Status: READY TO RUN ON MAC. Nothing in this pack is claimed as done until screenshots 1, 2, and 3 arrive back here.
Stage 1: stage1-fork-app-2026-11-04-01. Stage 2: stage2-fork-app-2026-11-04-01.
Order matters: app first, demo page last. Do not start with the demo page, and do not open the installed Ollama app from Applications.

## What this proves, in plain words
That in the real fork Ollama app on a Mac, with a tool-capable model, an enabled plugin tool is actually called by the model in chat, the result comes back, and the model answers using that result.

## Step 0: Checkout, stop if anything is missing
- Fork repo, branch plugin-seed, commit 4af35b9136bc03a02d9e177ea1bfdc771e89f397.
- Verify these files exist: app/tools/plugin.go, app/tools/plugin_test.go, app/ui/app/src/components/Plugins.tsx.
- If any is missing, stop and report the exact missing file. Do not substitute the stock app, and do not use the companion seed repo as a substitute for the fork.

## Step 1: Fork server, prove it is the fork
- Stop the stock Ollama server if it is running on 11434.
- Build and run the fork server from plugin-seed on 11434.
- Check: GET http://127.0.0.1:11434/api/plugins must return plugins.
- Why: stock Ollama does not have /api/plugins, so this check proves the server is the fork. If it returns 404, stop, the server is stock, not the fork.

## Step 2: Fork desktop app, production UI build, not dev mode
- In app/ui/app: npm install, then npm run build, with no new dependencies.
- In app: go generate ./..., then go run ./cmd/app.
- Do not open the installed Ollama app from Applications.
- Do not use the -dev flag in this run, because -dev fixes the API port to 3001.
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
