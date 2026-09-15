export type Service = {
  id: string;
  name: string;
  project: string;
  directory: string;
  command: string;
  port: number;
  url: string;
  framework: string;
  source: string;
  pid: number;
  birth: number;
  status: "running" | "starting" | "external" | "stopped";
  managed: boolean;
  cover: string;
  screenshotError: string;
  processName?: string;
  ports?: number[];
  discoveryReason?: string;
  discoveryDetail?: string;
  failure?: string;
  failureDetail?: string;
  env?: EnvVar[];
  health?: Health;
  updated: string;
};
export type EnvVar = { name: string; value: string; secret?: boolean };
export type Health = {
  process: "running" | "stopped" | "unknown";
  tcp: "reachable" | "unreachable" | "unknown";
  http: "healthy" | "failed" | "unavailable" | "unknown";
  httpStatus?: number;
  error?: string;
  checkedAt?: string;
};
export type ServiceInput = Pick<
  Service,
  "name" | "project" | "directory" | "command" | "port" | "url"
> & { env?: EnvVar[] };
export type Snapshot = {
  services: Service[];
  lastScan: string;
  scanError: string;
};
export type Session = {
  token: string;
  platform: string;
  browserAvailable: boolean;
  dataDirectory: string;
};
export type Language = "zh" | "en";
export type Settings = {
  language: Language;
  autoScan: boolean;
  scanIntervalSeconds: number;
  minScanIntervalSeconds: number;
  maxScanIntervalSeconds: number;
  includeDirectories: string[];
  excludeDirectories: string[];
  includeProcesses: string[];
  excludeProcesses: string[];
  includePorts: number[];
  excludePorts: number[];
  theme: "dark" | "light";
  accent: "lime" | "cyan" | "violet" | "amber";
  density: "comfortable" | "compact";
  serviceView: "cards" | "list";
  projectPrefs: ProjectPref[];
};
export type ProjectPref = {
  name: string;
  color: string;
  icon: string;
  order: number;
};
export type Log = { id: number; time: string; text: string };
export const active = (s: Service) => s.status !== "stopped";
export function filterServices(
  services: Service[],
  query: string,
  filter: string,
  project: string,
  prefs: ProjectPref[] = [],
) {
  const q = query.toLocaleLowerCase();
  const order = new Map(prefs.map((item) => [item.name, item.order]));
  return services
    .filter(
      (s) =>
        (!project || s.project === project) &&
        (filter === "all" ||
          (filter === "active" ? active(s) : s.status === "stopped")) &&
        `${s.name} ${s.project} ${s.directory} ${s.port} ${s.framework} ${s.processName ?? ""} ${(s.ports ?? []).join(" ")} ${s.discoveryDetail ?? ""}`
          .toLocaleLowerCase()
          .includes(q),
    )
    .sort(
      (a, b) =>
        (order.get(a.project) ?? 1000) - (order.get(b.project) ?? 1000) ||
        a.project.localeCompare(b.project) ||
        a.port - b.port,
    );
}
export function projectPref(prefs: ProjectPref[], name: string): ProjectPref {
  return (
    prefs.find((item) => item.name === name) || {
      name,
      color: "lime",
      icon: "folder",
      order: 1000,
    }
  );
}
export function servicePorts(service: Pick<Service, "port" | "ports">) {
  const ports = service.ports?.length ? service.ports : [service.port];
  return [...new Set(ports.filter((port) => port > 0))].sort((a, b) => a - b);
}
export function parseRuleLines(text: string) {
  return text
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean);
}
export function parsePortRules(text: string) {
  const seen = new Set<number>();
  const ports: number[] = [];
  for (const part of text.split(/[\s,;]+/)) {
    if (!part) continue;
    const port = Number(part);
    if (!Number.isInteger(port) || port < 1 || port > 65535 || seen.has(port)) {
      continue;
    }
    seen.add(port);
    ports.push(port);
  }
  return ports;
}
