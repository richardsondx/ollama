import { describe, expect, it } from "vitest";
import { stepNameForTool, parseToolArgs } from "./turnSteps";

describe("turn step names", () => {
  it("names open, tab, type, key, and check steps in plain words", () => {
    expect(stepNameForTool("computer.open_app", { app: "Safari" })).toBe(
      "Open Safari",
    );
    expect(
      stepNameForTool("browser.create_tab", { url: "https://yahoo.com" }),
    ).toBe("Open https://yahoo.com in a new tab");
    expect(
      stepNameForTool("computer.type_text", {
        app: "Safari",
        text: "yahoo.com",
      }),
    ).toBe("Type in Safari");
    expect(
      stepNameForTool("computer.press_key", { app: "Safari", key: "Enter" }),
    ).toBe("Press Enter in Safari");
    expect(
      stepNameForTool("computer.get_app_state", { app: "Safari" }),
    ).toBe("Check Safari");
    expect(stepNameForTool("browser.get_tab", {})).toBe("Check tab");
  });

  it("never puts typed text into a step name", () => {
    const name = stepNameForTool("computer.type_text", {
      app: "Safari",
      text: "secret-password",
    });
    expect(name).not.toContain("secret-password");
  });

  it("parses bad argument JSON as an empty map", () => {
    expect(parseToolArgs("not json")).toEqual({});
    expect(parseToolArgs("[]")).toEqual({});
  });
});
