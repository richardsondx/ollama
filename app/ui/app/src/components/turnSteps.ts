export type TurnStep = {
  rawName: string;
  stepName: string;
  argsJson: string;
  result: string | null;
};

export function isBrowserComputerTool(name: string): boolean {
  return name.startsWith("browser.") || name.startsWith("computer.");
}

export function parseToolArgs(argsJson: string): Record<string, unknown> {
  try {
    const parsed = JSON.parse(argsJson || "{}") as unknown;
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>;
    }
  } catch {
    // Fall through to an empty argument map.
  }
  return {};
}

function stringArg(
  args: Record<string, unknown>,
  key: string,
): string | undefined {
  const value = args[key];
  return typeof value === "string" && value ? value : undefined;
}

// Plain step name for one browser or computer tool call. Typed text is
// never included in a step name, so a password or other typed text can
// never leak into the transcript through a step label.
export function stepNameForTool(
  name: string,
  args: Record<string, unknown>,
): string {
  const app = stringArg(args, "app");
  const url = stringArg(args, "url");
  const key = stringArg(args, "key");
  const tool = stringArg(args, "tool");
  switch (name) {
    case "computer.open_app":
      return app ? `Open ${app}` : "Open app";
    case "browser.create_tab":
      return url ? `Open ${url} in a new tab` : "Open a new tab";
    case "computer.type_text":
      return app ? `Type in ${app}` : "Type";
    case "computer.press_key":
      if (app && key) {
        return `Press ${key} in ${app}`;
      }
      return key ? `Press ${key}` : "Press a key";
    case "computer.get_app_state":
      return app ? `Check ${app}` : "Check app";
    case "computer.get_state":
      return app ? `Check ${app}` : "Check computer";
    case "browser.get_tab":
      return "Check tab";
    case "browser.list_tabs":
      return "Check tabs";
    case "computer.list_apps":
      return "List apps";
    case "computer.click":
      return app ? `Click in ${app}` : "Click";
    case "browser.webmcp.list_tools":
      return "List page tools";
    case "browser.webmcp.execute_tool":
      return tool ? `Run page tool ${tool}` : "Run page tool";
    case "browser.search":
      return "Search the web";
    case "browser.open":
      return "Open page";
    case "browser.find":
      return "Find on page";
    default: {
      // Fallback for any other browser or computer tool: the last part
      // of the name, with underscores turned into spaces. The raw name
      // is still shown in the per-step details, behind a toggle.
      const last = name.split(".").pop() || name;
      const words = last.replace(/_/g, " ");
      return words.charAt(0).toUpperCase() + words.slice(1);
    }
  }
}
