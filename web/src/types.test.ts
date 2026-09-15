import { describe, expect, it } from "vitest";
import {
  filterServices,
  parsePortRules,
  parseRuleLines,
  servicePorts,
  type Service,
} from "./types";
const services = [
  {
    id: "a",
    name: "前端",
    project: "Shop",
    directory: "/shop",
    port: 3000,
    status: "running",
    framework: "vite",
  },
  {
    id: "b",
    name: "API",
    project: "Shop",
    directory: "/shop/api",
    port: 8080,
    status: "stopped",
    framework: "Go",
  },
  {
    id: "c",
    name: "Docs",
    project: "Docs",
    directory: "/docs",
    port: 4321,
    status: "external",
    framework: "astro",
  },
] as Service[];
describe("service filtering", () => {
  it("combines project, status and case-insensitive search", () =>
    expect(
      filterServices(services, "VITE", "active", "Shop").map((s) => s.id),
    ).toEqual(["a"]));
  it("searches ports and includes external listeners as active", () =>
    expect(
      filterServices(services, "4321", "active", "").map((s) => s.id),
    ).toEqual(["c"]));
  it("only shows stopped services in stopped filter", () =>
    expect(
      filterServices(services, "", "stopped", "").map((s) => s.id),
    ).toEqual(["b"]));
  it("searches extra ports and process names", () => {
    const extra = [
      {
        ...services[0],
        processName: "node.exe",
        ports: [3000, 9229],
      },
    ];
    expect(filterServices(extra, "9229", "all", "").map((s) => s.id)).toEqual([
      "a",
    ]);
    expect(
      filterServices(extra, "node.exe", "all", "").map((s) => s.id),
    ).toEqual(["a"]);
  });
});

describe("discovery rule parsing", () => {
  it("keeps unique absolute-looking lines and ports", () => {
    expect(parseRuleLines(" C:\\\\a \n\nC:\\\\b\n ")).toEqual([
      "C:\\\\a",
      "C:\\\\b",
    ]);
    expect(parsePortRules("3000, 9229 3000\n80; 70000 abc")).toEqual([
      3000, 9229, 80,
    ]);
  });
  it("falls back to the primary port list", () => {
    expect(servicePorts({ port: 3000, ports: [5173, 3000, 9229] })).toEqual([
      3000, 5173, 9229,
    ]);
    expect(servicePorts({ port: 8080 })).toEqual([8080]);
  });
});
