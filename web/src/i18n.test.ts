import { describe, expect, it } from "vitest";
import { createTranslator } from "./i18n";

describe("workspace translations", () => {
  it("renders both supported languages and interpolates values", () => {
    expect(createTranslator("zh")("scanEvery", { seconds: 15 })).toBe(
      "每 15 秒",
    );
    expect(createTranslator("en")("scanEvery", { seconds: 15 })).toBe(
      "Every 15s",
    );
  });

  it("falls back to Chinese for an unavailable key", () => {
    expect(createTranslator("en")("language")).toBe("Language");
    expect(createTranslator("en")("unknown-key")).toBe("unknown-key");
  });

  it("translates discovery rule explanations", () => {
    expect(createTranslator("zh")("discoveryReason_merged-ports")).toBe(
      "同一进程的多个端口已合并",
    );
    expect(createTranslator("en")("discoveryReason_merged-ports")).toBe(
      "Merged ports from the same process",
    );
    expect(createTranslator("zh")("alsoListening", { ports: ":9229" })).toBe(
      "还监听 :9229",
    );
    expect(createTranslator("zh")("startTimeout")).toContain("启动超时");
    expect(createTranslator("en")("retryStart")).toBe("Retry");
  });
});
