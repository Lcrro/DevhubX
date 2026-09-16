import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import App from "./App";
import { cleanup } from "@testing-library/react";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});
it("loads real state and searches service cards", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: string) => ({
      ok: true,
      json: async () =>
        input.endsWith("/session")
          ? { token: "test", platform: "windows", browserAvailable: true }
          : {
              services: [
                {
                  id: "1",
                  name: "My frontend",
                  project: "Store",
                  directory: "C:/store",
                  command: "npm run dev",
                  port: 3000,
                  url: "http://127.0.0.1:3000",
                  status: "running",
                  managed: true,
                  framework: "vite",
                },
              ],
              lastScan: "",
              scanError: "",
            },
    })),
  );
  render(<App />);
  expect(
    await screen.findByRole("heading", { name: "My frontend" }),
  ).toBeInTheDocument();
  fireEvent.change(screen.getByRole("textbox", { name: "Search services" }), {
    target: { value: "not-found" },
  });
  await waitFor(() =>
    expect(screen.getByText("No matching services")).toBeInTheDocument(),
  );
  expect(
    screen.queryByRole("heading", { name: "My frontend" }),
  ).not.toBeInTheDocument();
});

it("opens settings with discovery rule fields", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: string) => ({
      ok: true,
      json: async () =>
        String(input).endsWith("/session")
          ? { token: "test", platform: "windows", browserAvailable: true }
          : String(input).endsWith("/settings")
            ? {
                language: "zh",
                autoScan: true,
                scanIntervalSeconds: 15,
                minScanIntervalSeconds: 5,
                maxScanIntervalSeconds: 300,
                includeDirectories: [],
                excludeDirectories: [],
                includeProcesses: [],
                excludeProcesses: ["QQ.exe"],
                includePorts: [],
                excludePorts: [9229],
              }
            : { services: [], lastScan: "", scanError: "" },
    })),
  );
  render(<App />);
  await screen.findByText("每 15 秒");
  fireEvent.click(screen.getByRole("button", { name: "设置与帮助" }));
  expect(await screen.findByText("发现规则")).toBeInTheDocument();
  expect(screen.getByPlaceholderText("例如 QQ.exe")).toHaveValue("QQ.exe");
  expect(screen.getByPlaceholderText("例如 9229, 4301")).toHaveValue("9229");
});
