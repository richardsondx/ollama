import { useEffect, useMemo, useState } from "react";
import { API_BASE } from "@/lib/config";
import {
  ChevronRightIcon,
  MagnifyingGlassIcon,
} from "@heroicons/react/24/outline";

type PluginTool = {
  name: string;
  description: string;
};

type Plugin = {
  name: string;
  description: string;
  version: string;
  tools: PluginTool[];
  skills: string[];
  enabled: boolean;
};

type Tab = "plugins" | "skills" | "installed";

async function fetchPlugins(): Promise<Plugin[]> {
  const response = await fetch(`${API_BASE}/api/plugins`);
  if (!response.ok) {
    throw new Error(`Plugins server answered ${response.status}`);
  }
  const data = await response.json();
  return (data.plugins || []) as Plugin[];
}

export default function Plugins() {
  const [plugins, setPlugins] = useState<Plugin[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [tab, setTab] = useState<Tab>("plugins");
  const [query, setQuery] = useState("");
  const [selectedName, setSelectedName] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const load = async () => {
    setLoading(true);
    setError(null);
    try {
      const list = await fetchPlugins();
      setPlugins(list);
      if (!selectedName && list.length > 0) {
        setSelectedName(list[0].name);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not load plugins");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const visible = useMemo(() => {
    const q = query.trim().toLowerCase();
    let list = plugins;
    if (tab === "installed") {
      list = list.filter((p) => p.enabled);
    }
    if (q) {
      list = list.filter(
        (p) =>
          p.name.toLowerCase().includes(q) ||
          p.description.toLowerCase().includes(q),
      );
    }
    return list;
  }, [plugins, tab, query]);

  const selected =
    plugins.find((p) => p.name === selectedName) || visible[0] || null;

  const toggle = async (plugin: Plugin) => {
    setBusy(true);
    try {
      const action = plugin.enabled ? "disable" : "enable";
      const response = await fetch(
        `${API_BASE}/api/plugins/${plugin.name}/${action}`,
        { method: "POST" },
      );
      if (!response.ok) {
        throw new Error(`Could not ${action} ${plugin.name}`);
      }
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not change plugin");
    } finally {
      setBusy(false);
    }
  };

  const tabClass = (t: Tab) =>
    `rounded-full px-3 py-1.5 text-sm ${
      tab === t
        ? "bg-neutral-900 text-white dark:bg-white dark:text-black"
        : "text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
    }`;

  return (
    <div className="flex min-h-0 flex-1 bg-white text-neutral-900 dark:bg-neutral-900 dark:text-white">
      <aside className="flex w-72 flex-none flex-col border-r border-neutral-200 bg-neutral-50 dark:border-neutral-800 dark:bg-neutral-950/40">
        <div className="border-b border-neutral-200 px-4 py-4 dark:border-neutral-800">
          <h2 className="text-base font-medium">Plugins</h2>
          <div className="mt-3 flex items-center gap-2 rounded-full border border-neutral-200 bg-white px-3 py-2 dark:border-neutral-700 dark:bg-neutral-900">
            <MagnifyingGlassIcon className="h-4 w-4 text-neutral-400" />
            <input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Search plugins"
              className="w-full bg-transparent text-sm outline-none placeholder:text-neutral-400"
            />
          </div>
          <div className="mt-3 flex gap-1">
            <button className={tabClass("plugins")} onClick={() => setTab("plugins")}>
              Plugins
            </button>
            <button className={tabClass("skills")} onClick={() => setTab("skills")}>
              Skills
            </button>
            <button
              className={tabClass("installed")}
              onClick={() => setTab("installed")}
            >
              Installed
            </button>
          </div>
        </div>
        <div className="flex-1 overflow-y-auto p-2">
          {loading && (
            <p className="px-2 py-4 text-sm text-neutral-500">
              Loading plugins…
            </p>
          )}
          {!loading && visible.length === 0 && (
            <p className="px-2 py-4 text-sm text-neutral-500">
              {tab === "installed"
                ? "No plugins enabled yet. Open a plugin and enable it."
                : "No plugins found."}
            </p>
          )}
          {visible.map((plugin) => (
            <button
              key={plugin.name}
              onClick={() => {
                setSelectedName(plugin.name);
                if (tab === "skills") setTab("plugins");
              }}
              className={`flex w-full items-center gap-3 rounded-lg px-2 py-2 text-left hover:bg-neutral-100 dark:hover:bg-neutral-800 ${
                selected?.name === plugin.name
                  ? "bg-neutral-100 dark:bg-neutral-800"
                  : ""
              }`}
            >
              <span className="flex h-8 w-8 flex-none items-center justify-center rounded-lg bg-neutral-900 text-sm font-medium text-white dark:bg-white dark:text-black">
                {plugin.name.charAt(0).toUpperCase()}
              </span>
              <span className="min-w-0 flex-1">
                <span className="block truncate text-sm font-medium">
                  {plugin.name}
                </span>
                <span className="block truncate text-xs text-neutral-500">
                  {plugin.enabled ? "Enabled" : "Not enabled"} ·{" "}
                  {plugin.tools.length} tools
                </span>
              </span>
            </button>
          ))}
          {tab === "skills" && (
            <div className="mt-2 border-t border-neutral-200 px-2 pt-2 dark:border-neutral-800">
              {plugins.map((plugin) =>
                plugin.skills.map((skill) => (
                  <p key={skill} className="py-1 text-sm text-neutral-600 dark:text-neutral-300">
                    {skill}
                  </p>
                )),
              )}
            </div>
          )}
        </div>
      </aside>

      <section className="min-w-0 flex-1 overflow-y-auto px-8 py-6">
        {error && (
          <p className="mb-4 rounded-lg border border-neutral-200 bg-neutral-50 px-4 py-3 text-sm text-neutral-700 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-200">
            {error} The plugin server needs a plugin folder. Set
            OLLAMA_PLUGINS_DIR, restart Ollama, and open this page again.
          </p>
        )}
        {!selected && !loading && (
          <div>
            <h3 className="text-lg font-medium">No plugins yet</h3>
            <p className="mt-2 max-w-xl text-sm leading-6 text-neutral-600 dark:text-neutral-300">
              Add a plugin folder with a plugin.json manifest and a SKILL.md
              file, point OLLAMA_PLUGINS_DIR at its parent folder, and restart
              Ollama. This page lists what the server finds, it does not
              install plugins from the web.
            </p>
          </div>
        )}
        {selected && (
          <div className="max-w-3xl">
            <p className="text-sm text-neutral-500">
              Plugins &gt; {selected.name}
            </p>
            <div className="mt-4 flex flex-wrap items-start justify-between gap-4">
              <div className="flex items-start gap-4">
                <span className="flex h-12 w-12 flex-none items-center justify-center rounded-xl bg-neutral-900 text-lg font-medium text-white dark:bg-white dark:text-black">
                  {selected.name.charAt(0).toUpperCase()}
                </span>
                <div>
                  <h3 className="text-xl font-medium">{selected.name}</h3>
                  <p className="mt-1 text-sm text-neutral-600 dark:text-neutral-300">
                    {selected.description}
                  </p>
                  <p className="mt-1 text-xs text-neutral-500">
                    Version {selected.version || "0.0.0"} ·{" "}
                    {selected.enabled ? "Enabled" : "Not enabled"}
                  </p>
                </div>
              </div>
              <button
                disabled={busy}
                onClick={() => void toggle(selected)}
                className="rounded-full bg-black px-4 py-2 text-sm text-white disabled:opacity-50 dark:bg-white dark:text-black"
              >
                {selected.enabled ? "Disable" : "Enable"}
              </button>
            </div>

            <div className="mt-6 rounded-2xl bg-neutral-100 px-5 py-4 dark:bg-neutral-800">
              <p className="text-sm font-medium">Try it in chat</p>
              <p className="mt-1 text-sm leading-6 text-neutral-600 dark:text-neutral-300">
                Enabling a plugin lists its tools at /api/tools. Chat does not
                call those tools yet, so enabling alone does not change what
                the model can do in this app. That chat wiring is the next
                step, not part of this page.
              </p>
            </div>

            <h4 className="mt-8 text-base font-medium">Setup</h4>
            <div className="mt-3 overflow-hidden rounded-xl border border-neutral-200 dark:border-neutral-700">
              {selected.tools.length === 0 && (
                <p className="px-4 py-3 text-sm text-neutral-500">
                  This plugin declares no tools.
                </p>
              )}
              {selected.tools.map((tool) => (
                <div
                  key={tool.name}
                  className="flex items-center justify-between gap-4 border-b border-neutral-200 px-4 py-3 last:border-b-0 dark:border-neutral-700"
                >
                  <div>
                    <p className="text-sm font-medium">{tool.name}</p>
                    <p className="text-sm text-neutral-500">
                      {tool.description}
                    </p>
                  </div>
                  <span className="flex-none rounded-full bg-neutral-100 px-3 py-1 text-xs text-neutral-600 dark:bg-neutral-800 dark:text-neutral-300">
                    Tool
                  </span>
                </div>
              ))}
            </div>

            <h4 className="mt-8 text-base font-medium">Skills</h4>
            <div className="mt-3 overflow-hidden rounded-xl border border-neutral-200 dark:border-neutral-700">
              {selected.skills.map((skill) => (
                <div
                  key={skill}
                  className="flex items-center justify-between px-4 py-3"
                >
                  <p className="text-sm">{skill}</p>
                  <ChevronRightIcon className="h-4 w-4 text-neutral-400" />
                </div>
              ))}
            </div>

            <h4 className="mt-8 text-base font-medium">About</h4>
            <dl className="mt-3 divide-y divide-neutral-200 rounded-xl border border-neutral-200 dark:divide-neutral-700 dark:border-neutral-700">
              <div className="flex justify-between px-4 py-3 text-sm">
                <dt className="text-neutral-500">Plugin</dt>
                <dd>{selected.name}</dd>
              </div>
              <div className="flex justify-between px-4 py-3 text-sm">
                <dt className="text-neutral-500">Version</dt>
                <dd>{selected.version || "0.0.0"}</dd>
              </div>
              <div className="flex justify-between px-4 py-3 text-sm">
                <dt className="text-neutral-500">Tools</dt>
                <dd>{selected.tools.length}</dd>
              </div>
              <div className="flex justify-between px-4 py-3 text-sm">
                <dt className="text-neutral-500">Status</dt>
                <dd>{selected.enabled ? "Enabled" : "Not enabled"}</dd>
              </div>
            </dl>
          </div>
        )}
      </section>
    </div>
  );
}
