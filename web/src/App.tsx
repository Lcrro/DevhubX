import { useCallback, useEffect, useRef, useState } from "react";
import type { FormEvent, ReactNode } from "react";
import {
  Activity,
  ArrowUpRight,
  Camera,
  Check,
  ChevronRight,
  CircleHelp,
  Code2,
  Folder,
  FolderOpen,
  LayoutGrid,
  LayoutList,
  Loader2,
  Plus,
  Radio,
  RefreshCw,
  Search,
  Settings2,
  ShieldCheck,
  Globe,
  Database,
  Upload,
  Archive,
  Square,
  Terminal,
  Trash2,
  X,
  Play,
  Pause,
  Download,
  Eraser,
  Server,
  Pencil,
  Copy,
} from "lucide-react";
import { api, session } from "./api";
import {
  active,
  filterServices,
  parsePortRules,
  parseRuleLines,
  projectPref,
  servicePorts,
} from "./types";
import type {
  Service,
  ServiceInput,
  Snapshot,
  Session,
  Log,
  Settings,
  Language,
  ProjectPref,
} from "./types";
import { createTranslator } from "./i18n";

const errorText = (e: unknown) => (e instanceof Error ? e.message : String(e));
const defaultSettings: Settings = {
  language: "zh",
  autoScan: true,
  scanIntervalSeconds: 10,
  minScanIntervalSeconds: 5,
  maxScanIntervalSeconds: 300,
  includeDirectories: [],
  excludeDirectories: [],
  includeProcesses: [],
  excludeProcesses: [],
  includePorts: [],
  excludePorts: [],
  theme: "dark",
  accent: "lime",
  density: "comfortable",
  serviceView: "cards",
  projectPrefs: [],
};

function ProjectIcon({ name, size }: { name: string; size: number }) {
  if (name === "code") return <Code2 size={size} />;
  if (name === "server") return <Server size={size} />;
  if (name === "globe") return <Globe size={size} />;
  if (name === "database") return <Database size={size} />;
  return <Folder size={size} />;
}

function upsertPref(prefs: ProjectPref[], next: ProjectPref) {
  const rest = prefs.filter((item) => item.name !== next.name);
  return [...rest, next];
}

function rewriteUrlPort(url: string, from: number, to: number) {
  if (!url.trim()) return "";
  return url.replace(`:${from}`, `:${to}`);
}

function failureMessage(
  s: Service,
  t: (key: string, vars?: Record<string, string | number>) => string,
) {
  if (s.failure === "start-timeout") return t("startTimeout");
  if (s.failure === "start-exit") {
    return s.failureDetail
      ? `${t("startExit")} · ${s.failureDetail}`
      : t("startExit");
  }
  if (s.health?.error === "process-no-port") return t("processNoPort");
  if (s.health?.http === "failed" && s.health.error) {
    return `${t("httpHealth")} · ${s.health.error}`;
  }
  return s.health?.error || s.failure || "";
}

function Modal({
  title,
  children,
  close,
  wide = false,
  closeLabel = "关闭",
}: {
  title: string;
  children: ReactNode;
  close: () => void;
  wide?: boolean;
  closeLabel?: string;
}) {
  const ref = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    ref.current?.showModal();
    return () => ref.current?.close();
  }, []);
  return (
    <dialog
      ref={ref}
      aria-label={title}
      className={wide ? "modal wide" : "modal"}
      onCancel={close}
      onClick={(e) => {
        if (e.target === e.currentTarget) close();
      }}
    >
      <div className="modal-head">
        <h2>{title}</h2>
        <button className="icon-button" aria-label={closeLabel} onClick={close}>
          <X size={20} />
        </button>
      </div>
      {children}
    </dialog>
  );
}

function Editor({
  service,
  close,
  saved,
  platform,
  language,
}: {
  service?: Service;
  close: () => void;
  saved: () => void;
  platform?: string;
  language: Language;
}) {
  const t = createTranslator(language);
  const [form, setForm] = useState<ServiceInput>(
    service
      ? {
          name: service.name,
          project: service.project,
          directory: service.directory,
          command: service.command,
          port: service.port,
          url: service.url,
          env: service.env || [],
        }
      : {
          name: "",
          project: "",
          directory: "",
          command: "",
          port: 3000,
          url: "",
          env: [],
        },
  );
  const [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  const field = (name: keyof ServiceInput, value: string | number) =>
    setForm((f) => ({ ...f, [name]: value }));
  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await api(
        `/services${service ? `/${service.id}` : ""}`,
        service ? "PUT" : "POST",
        form,
      );
      saved();
      close();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal
      title={service ? t("editService") : t("addLocal")}
      close={close}
      closeLabel={t("close")}
    >
      <form onSubmit={submit}>
        <p className="muted">{t("saveLocation")}</p>
        <div className="form-row">
          <label>
            {t("serviceName")}
            <input
              autoFocus
              required
              maxLength={100}
              placeholder={t("serviceNamePlaceholder")}
              value={form.name}
              onChange={(e) => field("name", e.target.value)}
            />
          </label>
          <label>
            {t("projectName")}
            <input
              placeholder={t("projectPlaceholder")}
              value={form.project}
              onChange={(e) => field("project", e.target.value)}
            />
          </label>
        </div>
        <label>
          {t("directory")}
          <input
            required
            placeholder={
              platform === "windows"
                ? "C:\\Projects\\my-app"
                : "/Users/me/projects/my-app"
            }
            value={form.directory}
            onChange={(e) => field("directory", e.target.value)}
          />
          <small>{t("directoryHelp")}</small>
        </label>
        <label>
          {t("launchCommand")}
          <input
            placeholder="npm run dev"
            value={form.command}
            onChange={(e) => field("command", e.target.value)}
          />
          <small>
            {platform === "windows" ? t("powershellHelp") : t("shellHelp")}{" "}
            {t("pasteHelp")}
          </small>
        </label>
        <div className="form-row">
          <label>
            {t("port")}
            <input
              required
              type="number"
              min="1"
              max="65535"
              value={form.port}
              onChange={(e) => field("port", Number(e.target.value))}
            />
          </label>
          <label>
            {t("webAddress")}
            <input
              placeholder={`http://127.0.0.1:${form.port}`}
              value={form.url}
              onChange={(e) => field("url", e.target.value)}
            />
          </label>
        </div>
        <div className="env-editor">
          <span>{t("envVars")}</span>
          <small>{t("envVarsHelp")}</small>
          {(form.env || []).map((item, index) => (
            <div className="env-row" key={index}>
              <input
                placeholder={t("envName")}
                value={item.name}
                onChange={(e) => {
                  const env = [...(form.env || [])];
                  env[index] = { ...env[index], name: e.target.value };
                  setForm((f) => ({ ...f, env }));
                }}
              />
              <input
                placeholder={t("envValue")}
                type={item.secret ? "password" : "text"}
                value={item.value}
                onChange={(e) => {
                  const env = [...(form.env || [])];
                  env[index] = { ...env[index], value: e.target.value };
                  setForm((f) => ({ ...f, env }));
                }}
              />
              <label className="secret-flag">
                <input
                  type="checkbox"
                  checked={!!item.secret}
                  onChange={(e) => {
                    const env = [...(form.env || [])];
                    env[index] = { ...env[index], secret: e.target.checked };
                    setForm((f) => ({ ...f, env }));
                  }}
                />
                {t("secretValue")}
              </label>
              <button
                type="button"
                className="icon-button"
                aria-label={t("remove")}
                onClick={() =>
                  setForm((f) => ({
                    ...f,
                    env: (f.env || []).filter((_, i) => i !== index),
                  }))
                }
              >
                <X size={16} />
              </button>
            </div>
          ))}
          <button
            type="button"
            className="button"
            onClick={() =>
              setForm((f) => ({
                ...f,
                env: [...(f.env || []), { name: "", value: "", secret: false }],
              }))
            }
          >
            {t("addEnv")}
          </button>
        </div>
        {service && servicePorts(service).length > 1 && (
          <label>
            {t("primaryWebPort")}
            <select
              value={form.port}
              onChange={(e) => {
                const port = Number(e.target.value);
                setForm((current) => ({
                  ...current,
                  port,
                  url: rewriteUrlPort(current.url, current.port, port),
                }));
              }}
            >
              {servicePorts(service).map((port) => (
                <option key={port} value={port}>
                  :{port}
                </option>
              ))}
            </select>
          </label>
        )}
        <p className="hint">{t("portHelp")}</p>
        {error && (
          <p role="alert" className="error">
            {error}
          </p>
        )}
        <div className="modal-actions">
          <button type="button" className="button" onClick={close}>
            {t("cancel")}
          </button>
          <button className="button primary" disabled={busy}>
            {busy ? t("saving") : t("saveService")}
          </button>
        </div>
      </form>
    </Modal>
  );
}

function Logs({
  service,
  close,
  language,
}: {
  service: Service;
  close: () => void;
  language: Language;
}) {
  const t = createTranslator(language);
  const [logs, setLogs] = useState<Log[]>([]),
    [error, setError] = useState("");
  const [follow, setFollow] = useState(true),
    [paused, setPaused] = useState(false),
    [copied, setCopied] = useState(false),
    [query, setQuery] = useState("");
  const pausedRef = useRef(false);
  const afterRef = useRef(0);
  const end = useRef<HTMLDivElement>(null);
  useEffect(() => {
    let gone = false;
    const read = async () => {
      if (pausedRef.current) return;
      try {
        const next = await api<Log[]>(
          `/services/${service.id}/logs?after=${afterRef.current}`,
        );
        if (!gone && next.length) {
          afterRef.current = next[next.length - 1].id;
          setLogs((old) => [...old, ...next].slice(-500));
        }
        if (!gone) setError("");
      } catch (e) {
        if (!gone) setError(errorText(e));
      }
    };
    void read();
    const t = setInterval(read, 1500);
    return () => {
      gone = true;
      clearInterval(t);
    };
  }, [service.id]);
  useEffect(() => {
    if (follow) end.current?.scrollIntoView({ block: "nearest" });
  }, [logs, follow]);
  const visibleLogs = query.trim()
    ? logs.filter((l) =>
        l.text.toLocaleLowerCase().includes(query.trim().toLocaleLowerCase()),
      )
    : logs;
  function togglePaused(value: boolean) {
    pausedRef.current = value;
    setPaused(value);
  }
  return (
    <Modal
      title={`${service.name} · ${t("logs")}`}
      close={close}
      wide
      closeLabel={t("close")}
    >
      <div className="log-toolbar">
        <code>{service.command || t("externalService")}</code>
        <label className="log-search">
          <Search size={15} />
          <input
            aria-label={t("searchLogs")}
            placeholder={t("searchLogsPlaceholder")}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </label>
        <label className="check-label">
          <input
            type="checkbox"
            checked={follow}
            onChange={(e) => setFollow(e.target.checked)}
          />
          {t("followLogs")}
        </label>
        <button
          className="log-control"
          aria-label={paused ? t("resumeLogs") : t("pauseLogs")}
          title={paused ? t("resumeLogs") : t("pauseLogs")}
          onClick={() => togglePaused(!paused)}
        >
          {paused ? <Play size={15} /> : <Pause size={15} />}
          {paused ? t("resumeLogs") : t("pauseLogs")}
        </button>
        <button
          className="icon-button"
          aria-label={t("copyLogs")}
          title={t("copyLogs")}
          onClick={async () => {
            try {
              await navigator.clipboard.writeText(
                logs.map((l) => l.text).join(""),
              );
              setCopied(true);
            } catch {
              setError(t("copyFailed"));
            }
          }}
        >
          {copied ? <Check size={16} /> : <Copy size={16} />}
        </button>
        <button
          className="icon-button"
          aria-label={t("exportLogs")}
          title={t("exportLogs")}
          disabled={!logs.length}
          onClick={() => {
            const text = logs
              .map((l) => `[${new Date(l.time).toISOString()}] ${l.text}`)
              .join("");
            const url = URL.createObjectURL(
              new Blob([text], { type: "text/plain;charset=utf-8" }),
            );
            const link = document.createElement("a");
            link.href = url;
            link.download = `${service.name.replace(/[^\w.-]+/g, "_")}-logs.txt`;
            link.click();
            URL.revokeObjectURL(url);
          }}
        >
          <Download size={16} />
        </button>
        <button
          className="icon-button danger-icon"
          aria-label={t("clearLogs")}
          title={t("clearLogs")}
          disabled={!logs.length}
          onClick={async () => {
            if (!window.confirm(t("clearLogsConfirm", { name: service.name })))
              return;
            try {
              await api(`/services/${service.id}/logs`, "DELETE", {});
              setLogs([]);
              afterRef.current = 0;
              setError("");
            } catch (e) {
              setError(errorText(e));
            }
          }}
        >
          <Eraser size={16} />
        </button>
      </div>
      {!service.managed && <p className="hint">{t("externalLogsHint")}</p>}
      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
      <div className="logs" aria-label={t("logs")}>
        {visibleLogs.length ? (
          visibleLogs.map((l) => (
            <div key={l.id}>
              <time>{new Date(l.time).toLocaleTimeString()}</time>
              <pre>{l.text.replace(/\x1b\[[0-9;]*[a-zA-Z]/g, "")}</pre>
            </div>
          ))
        ) : (
          <p>{logs.length ? t("noMatchingLogs") : t("noLogs")}</p>
        )}
        <div ref={end} />
      </div>
    </Modal>
  );
}

function SettingsModal({
  settings,
  info,
  language,
  projects,
  close,
  saved,
}: {
  settings: Settings;
  info?: Session;
  language: Language;
  projects: string[];
  close: () => void;
  saved: (settings: Settings) => void;
}) {
  const t = createTranslator(language);
  const [draft, setDraft] = useState(settings);
  const [includeDirectories, setIncludeDirectories] = useState(
    settings.includeDirectories.join("\n"),
  );
  const [excludeDirectories, setExcludeDirectories] = useState(
    settings.excludeDirectories.join("\n"),
  );
  const [includeProcesses, setIncludeProcesses] = useState(
    settings.includeProcesses.join("\n"),
  );
  const [excludeProcesses, setExcludeProcesses] = useState(
    settings.excludeProcesses.join("\n"),
  );
  const [includePorts, setIncludePorts] = useState(
    settings.includePorts.join("\n"),
  );
  const [excludePorts, setExcludePorts] = useState(
    settings.excludePorts.join("\n"),
  );
  const [includeSecrets, setIncludeSecrets] = useState(false);
  const [includeLogs, setIncludeLogs] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      const value = await api<Settings>("/settings", "PUT", {
        language: draft.language,
        autoScan: draft.autoScan,
        scanIntervalSeconds: draft.scanIntervalSeconds,
        includeDirectories: parseRuleLines(includeDirectories),
        excludeDirectories: parseRuleLines(excludeDirectories),
        includeProcesses: parseRuleLines(includeProcesses),
        excludeProcesses: parseRuleLines(excludeProcesses),
        includePorts: parsePortRules(includePorts),
        excludePorts: parsePortRules(excludePorts),
        theme: draft.theme,
        accent: draft.accent,
        density: draft.density,
        serviceView: draft.serviceView,
        projectPrefs: draft.projectPrefs,
      });
      saved({
        ...settings,
        ...value,
        language: value.language || draft.language,
        includeDirectories: value.includeDirectories || [],
        excludeDirectories: value.excludeDirectories || [],
        includeProcesses: value.includeProcesses || [],
        excludeProcesses: value.excludeProcesses || [],
        includePorts: value.includePorts || [],
        excludePorts: value.excludePorts || [],
        theme: value.theme || draft.theme,
        accent: value.accent || draft.accent,
        density: value.density || draft.density,
        serviceView: value.serviceView || draft.serviceView,
        projectPrefs: value.projectPrefs || [],
      });
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }
  async function exportConfig() {
    try {
      const bundle = await api<unknown>("/export", "POST", {
        logs: includeLogs,
        secrets: includeSecrets,
      });
      const blob = new Blob([JSON.stringify(bundle, null, 2)], {
        type: "application/json",
      });
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = "devhub-export.json";
      link.click();
      URL.revokeObjectURL(url);
    } catch (e) {
      setError(errorText(e));
    }
  }
  async function importConfig(file?: File) {
    if (!file) return;
    try {
      const bundle = JSON.parse(await file.text());
      const result = await api<{ imported: number }>("/import", "POST", bundle);
      saved({ ...settings, ...draft });
      setError("");
      window.alert(t("importedOk", { count: result.imported || 0 }));
    } catch (e) {
      setError(errorText(e));
    }
  }
  async function backupNow() {
    try {
      const result = await api<{ path: string }>("/backup", "POST", {});
      window.alert(t("backupSaved", { path: result.path }));
    } catch (e) {
      setError(errorText(e));
    }
  }
  return (
    <Modal
      title={t("localWorkspace")}
      close={close}
      closeLabel={t("close")}
      wide
    >
      <form onSubmit={submit}>
        <div className="settings-section">
          <h3>
            <Settings2 size={18} />
            {t("appearance")}
          </h3>
          <label className="settings-control">
            <span>{t("language")}</span>
            <select
              value={draft.language}
              onChange={(e) =>
                setDraft((v) => ({
                  ...v,
                  language: e.target.value as Language,
                }))
              }
            >
              <option value="zh">{t("chinese")}</option>
              <option value="en">{t("english")}</option>
            </select>
          </label>
          <label className="settings-control">
            <span>{t("theme")}</span>
            <select
              value={draft.theme}
              onChange={(e) =>
                setDraft((v) => ({
                  ...v,
                  theme: e.target.value as Settings["theme"],
                }))
              }
            >
              <option value="dark">{t("themeDark")}</option>
              <option value="light">{t("themeLight")}</option>
            </select>
          </label>
          <label className="settings-control">
            <span>{t("accent")}</span>
            <select
              value={draft.accent}
              onChange={(e) =>
                setDraft((v) => ({
                  ...v,
                  accent: e.target.value as Settings["accent"],
                }))
              }
            >
              <option value="lime">{t("accentLime")}</option>
              <option value="cyan">{t("accentCyan")}</option>
              <option value="violet">{t("accentViolet")}</option>
              <option value="amber">{t("accentAmber")}</option>
            </select>
          </label>
          <label className="settings-control">
            <span>{t("density")}</span>
            <select
              value={draft.density}
              onChange={(e) =>
                setDraft((v) => ({
                  ...v,
                  density: e.target.value as Settings["density"],
                }))
              }
            >
              <option value="comfortable">{t("densityComfortable")}</option>
              <option value="compact">{t("densityCompact")}</option>
            </select>
          </label>
          <label className="settings-control">
            <span>{t("serviceView")}</span>
            <select
              value={draft.serviceView}
              onChange={(e) =>
                setDraft((v) => ({
                  ...v,
                  serviceView: e.target.value as Settings["serviceView"],
                }))
              }
            >
              <option value="cards">{t("viewCards")}</option>
              <option value="list">{t("viewList")}</option>
            </select>
          </label>
          <p className="hint">{t("shortcutsHint")}</p>
        </div>
        <div className="settings-section">
          <h3>
            <Radio size={18} />
            {t("scanSettings")}
          </h3>
          <label className="settings-control check-setting">
            <span>{t("enableScan")}</span>
            <input
              type="checkbox"
              checked={draft.autoScan}
              onChange={(e) =>
                setDraft((v) => ({ ...v, autoScan: e.target.checked }))
              }
            />
          </label>
          <label className="settings-control interval-setting">
            <span>{t("interval")}</span>
            <div className="interval-input">
              <input
                type="number"
                min={draft.minScanIntervalSeconds}
                max={draft.maxScanIntervalSeconds}
                value={draft.scanIntervalSeconds}
                onChange={(e) =>
                  setDraft((v) => ({
                    ...v,
                    scanIntervalSeconds: Number(e.target.value),
                  }))
                }
              />
              <span>{t("seconds")}</span>
            </div>
          </label>
          <input
            className="interval-range"
            type="range"
            min={draft.minScanIntervalSeconds}
            max={draft.maxScanIntervalSeconds}
            value={draft.scanIntervalSeconds}
            onChange={(e) =>
              setDraft((v) => ({
                ...v,
                scanIntervalSeconds: Number(e.target.value),
              }))
            }
            aria-label={t("interval")}
          />
          <p className="hint">
            {t("intervalHint", {
              min: draft.minScanIntervalSeconds,
              max: draft.maxScanIntervalSeconds,
            })}
          </p>
        </div>
        <div className="settings-section">
          <h3>
            <Radio size={18} />
            {t("discoveryRules")}
          </h3>
          <p>{t("discoveryRulesHint")}</p>
          <div className="rules-grid">
            <label>
              {t("includeDirectories")}
              <textarea
                rows={3}
                placeholder={t("includeDirectoriesPlaceholder")}
                value={includeDirectories}
                onChange={(e) => setIncludeDirectories(e.target.value)}
              />
              <small>{t("directoryRulesHelp")}</small>
            </label>
            <label>
              {t("excludeDirectories")}
              <textarea
                rows={3}
                placeholder={t("excludeDirectoriesPlaceholder")}
                value={excludeDirectories}
                onChange={(e) => setExcludeDirectories(e.target.value)}
              />
              <small>{t("directoryRulesHelp")}</small>
            </label>
            <label>
              {t("includeProcesses")}
              <textarea
                rows={3}
                placeholder={t("includeProcessesPlaceholder")}
                value={includeProcesses}
                onChange={(e) => setIncludeProcesses(e.target.value)}
              />
              <small>{t("processRulesHelp")}</small>
            </label>
            <label>
              {t("excludeProcesses")}
              <textarea
                rows={3}
                placeholder={t("excludeProcessesPlaceholder")}
                value={excludeProcesses}
                onChange={(e) => setExcludeProcesses(e.target.value)}
              />
              <small>{t("processRulesHelp")}</small>
            </label>
            <label>
              {t("includePorts")}
              <textarea
                rows={3}
                placeholder={t("includePortsPlaceholder")}
                value={includePorts}
                onChange={(e) => setIncludePorts(e.target.value)}
              />
              <small>{t("portRulesHelp")}</small>
            </label>
            <label>
              {t("excludePorts")}
              <textarea
                rows={3}
                placeholder={t("excludePortsPlaceholder")}
                value={excludePorts}
                onChange={(e) => setExcludePorts(e.target.value)}
              />
              <small>{t("portRulesHelp")}</small>
            </label>
          </div>
        </div>
        <div className="settings-section">
          <h3>
            <ShieldCheck size={18} />
            {t("connectionStorage")}
          </h3>
          <p>{t("connectionStorageHint")}</p>
          <code className="path-block">
            {info?.dataDirectory || t("notConnected")}
          </code>
        </div>
        <div className="settings-section">
          <h3>
            <Radio size={18} />
            {t("discoveryScope")}
          </h3>
          <p>{t("discoveryHint", { seconds: draft.scanIntervalSeconds })}</p>
          <p>{t("noCommandGuess")}</p>
        </div>
        <div className="settings-section">
          <h3>
            <Camera size={18} />
            {t("webCovers")}
          </h3>
          <p>
            {info?.browserAvailable ? t("browserReady") : t("browserMissing")}{" "}
            {t("coverPrivacy")}
          </p>
        </div>
        <div className="settings-section">
          <h3>
            <Terminal size={18} />
            {t("processLogs")}
          </h3>
          <p>{t("processLogsHint")}</p>
        </div>
        {projects.length > 0 && (
          <div className="settings-section">
            <h3>
              <Folder size={18} />
              {t("projectAppearance")}
            </h3>
            <p>{t("projectAppearanceHint")}</p>
            {projects.map((name) => {
              const pref = projectPref(draft.projectPrefs, name);
              return (
                <div className="project-pref-row" key={name}>
                  <strong>{name}</strong>
                  <select
                    aria-label={`${name} ${t("projectColor")}`}
                    value={pref.color}
                    onChange={(e) =>
                      setDraft((v) => ({
                        ...v,
                        projectPrefs: upsertPref(v.projectPrefs, {
                          ...pref,
                          color: e.target.value,
                        }),
                      }))
                    }
                  >
                    <option value="lime">{t("accentLime")}</option>
                    <option value="cyan">{t("accentCyan")}</option>
                    <option value="violet">{t("accentViolet")}</option>
                    <option value="amber">{t("accentAmber")}</option>
                    <option value="rose">Rose</option>
                    <option value="gray">Gray</option>
                  </select>
                  <select
                    aria-label={`${name} ${t("projectIcon")}`}
                    value={pref.icon}
                    onChange={(e) =>
                      setDraft((v) => ({
                        ...v,
                        projectPrefs: upsertPref(v.projectPrefs, {
                          ...pref,
                          icon: e.target.value,
                        }),
                      }))
                    }
                  >
                    <option value="folder">folder</option>
                    <option value="code">code</option>
                    <option value="server">server</option>
                    <option value="globe">globe</option>
                    <option value="database">database</option>
                  </select>
                  <input
                    type="number"
                    aria-label={`${name} ${t("projectOrder")}`}
                    value={pref.order}
                    onChange={(e) =>
                      setDraft((v) => ({
                        ...v,
                        projectPrefs: upsertPref(v.projectPrefs, {
                          ...pref,
                          order: Number(e.target.value),
                        }),
                      }))
                    }
                  />
                </div>
              );
            })}
          </div>
        )}
        <div className="settings-section">
          <h3>
            <Archive size={18} />
            {t("backupRestore")}
          </h3>
          <p>{t("backupRestoreHint")}</p>
          <label className="settings-control check-setting">
            <span>{t("includeSecrets")}</span>
            <input
              type="checkbox"
              checked={includeSecrets}
              onChange={(e) => setIncludeSecrets(e.target.checked)}
            />
          </label>
          <label className="settings-control check-setting">
            <span>{t("includeLogs")}</span>
            <input
              type="checkbox"
              checked={includeLogs}
              onChange={(e) => setIncludeLogs(e.target.checked)}
            />
          </label>
          <div className="backup-actions">
            <button
              type="button"
              className="button"
              onClick={() => void exportConfig()}
            >
              {t("exportConfig")}
            </button>
            <label className="button">
              {t("importConfig")}
              <input
                type="file"
                accept="application/json"
                hidden
                onChange={(e) => void importConfig(e.target.files?.[0])}
              />
            </label>
            <button
              type="button"
              className="button"
              onClick={() => void backupNow()}
            >
              {t("backupNow")}
            </button>
          </div>
        </div>
        {error && (
          <p className="error" role="alert">
            {error || t("invalidSettings")}
          </p>
        )}
        <div className="modal-actions">
          <button type="button" className="button" onClick={close}>
            {t("cancel")}
          </button>
          <button className="button primary" disabled={busy}>
            {busy ? t("saving") : t("settingsSave")}
          </button>
        </div>
      </form>
    </Modal>
  );
}

export default function App() {
  const [snapshot, setSnapshot] = useState<Snapshot>({
    services: [],
    lastScan: "",
    scanError: "",
  });
  const [info, setInfo] = useState<Session>(),
    [loading, setLoading] = useState(true),
    [error, setError] = useState("");
  const [preferences, setPreferences] = useState<Settings>(defaultSettings);
  const language = preferences.language;
  const t = createTranslator(language);
  const [query, setQuery] = useState(""),
    [filter, setFilter] = useState("all"),
    [project, setProject] = useState("");
  const [busy, setBusy] = useState(""),
    [editor, setEditor] = useState<Service | "new">(),
    [logs, setLogs] = useState<Service>(),
    [settings, setSettings] = useState(false);
  const [confirm, setConfirm] = useState<{
      service: Service;
      action: "stop" | "delete";
    }>(),
    [notice, setNotice] = useState("");
  const refresh = useCallback(async () => {
    try {
      setSnapshot(await api<Snapshot>("/services"));
      setError("");
    } catch (e) {
      setError(errorText(e));
    } finally {
      setLoading(false);
    }
  }, []);
  useEffect(() => {
    let gone = false;
    void session()
      .then((value) => {
        if (!gone) {
          setInfo(value);
          void api<Partial<Settings>>("/settings")
            .then((stored) => {
              if (
                !gone &&
                stored.language &&
                (stored.language === "zh" || stored.language === "en")
              ) {
                setPreferences((current) => ({
                  ...current,
                  ...stored,
                  language: stored.language as Language,
                  includeDirectories:
                    stored.includeDirectories ?? current.includeDirectories,
                  excludeDirectories:
                    stored.excludeDirectories ?? current.excludeDirectories,
                  includeProcesses:
                    stored.includeProcesses ?? current.includeProcesses,
                  excludeProcesses:
                    stored.excludeProcesses ?? current.excludeProcesses,
                  includePorts: stored.includePorts ?? current.includePorts,
                  excludePorts: stored.excludePorts ?? current.excludePorts,
                  theme: stored.theme || current.theme,
                  accent: stored.accent || current.accent,
                  density: stored.density || current.density,
                  serviceView: stored.serviceView || current.serviceView,
                  projectPrefs: stored.projectPrefs ?? current.projectPrefs,
                }));
              }
            })
            .catch(() => undefined);
          void refresh();
        }
      })
      .catch((e) => {
        setError(errorText(e));
        setLoading(false);
      });
    const t = setInterval(refresh, 3000);
    return () => {
      gone = true;
      clearInterval(t);
    };
  }, [refresh]);
  useEffect(() => {
    if (!notice) return;
    const t = setTimeout(() => setNotice(""), 5000);
    return () => clearTimeout(t);
  }, [notice]);
  const projects = [...new Set(snapshot.services.map((s) => s.project))].sort(
    (a, b) =>
      (projectPref(preferences.projectPrefs, a).order || 1000) -
        (projectPref(preferences.projectPrefs, b).order || 1000) ||
      a.localeCompare(b),
  );
  const shown = filterServices(
    snapshot.services,
    query,
    filter,
    project,
    preferences.projectPrefs,
  );
  const searchRef = useRef<HTMLInputElement>(null);
  useEffect(() => {
    document.documentElement.dataset.theme = preferences.theme;
    document.documentElement.dataset.accent = preferences.accent;
    document.documentElement.dataset.density = preferences.density;
    document.documentElement.lang = language === "en" ? "en" : "zh-CN";
  }, [preferences.theme, preferences.accent, preferences.density, language]);
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      const target = e.target as HTMLElement | null;
      const typing =
        target &&
        (target.tagName === "INPUT" ||
          target.tagName === "TEXTAREA" ||
          target.tagName === "SELECT" ||
          target.isContentEditable);
      if (e.key === "Escape") {
        setSettings(false);
        setEditor(undefined);
        setLogs(undefined);
        return;
      }
      if (typing) return;
      if (e.key === "/") {
        e.preventDefault();
        searchRef.current?.focus();
      } else if (e.key === "n" || e.key === "N") {
        setEditor("new");
      } else if (e.key === "r" || e.key === "R") {
        void scan();
      } else if (e.key === "l" || e.key === "L") {
        setPreferences((v) => ({
          ...v,
          serviceView: v.serviceView === "list" ? "cards" : "list",
        }));
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });
  const running = snapshot.services.filter(active).length;
  async function action(s: Service, name: string, confirmed = false) {
    if (name === "delete" || (name === "stop" && !s.managed && !confirmed)) {
      setConfirm({ service: s, action: name as "stop" | "delete" });
      return;
    }
    setBusy(s.id);
    try {
      await api(`/services/${s.id}/${name}`, "POST", { confirm: confirmed });
      if (name === "capture") setNotice(t("coverQueued"));
      await refresh();
    } catch (e) {
      setNotice(errorText(e));
    } finally {
      setBusy("");
    }
  }
  async function retry(s: Service) {
    setBusy(s.id);
    try {
      if (active(s)) {
        await api(`/services/${s.id}/stop`, "POST", {});
      }
      await api(`/services/${s.id}/start`, "POST", {});
      await refresh();
    } catch (e) {
      setNotice(errorText(e));
    } finally {
      setBusy("");
    }
  }
  async function confirmAction() {
    if (!confirm) return;
    const { service: s, action: a } = confirm;
    setConfirm(undefined);
    if (a === "stop") {
      await action(s, a, true);
      return;
    }
    setBusy(s.id);
    try {
      await api(`/services/${s.id}`, "DELETE");
      await refresh();
    } catch (e) {
      setNotice(errorText(e));
    } finally {
      setBusy("");
    }
  }
  async function scan() {
    setBusy("scan");
    try {
      setSnapshot(await api<Snapshot>("/scan", "POST"));
      setNotice(t("scanComplete"));
    } catch (e) {
      setNotice(errorText(e));
    } finally {
      setBusy("");
    }
  }
  return (
    <div className="app-shell">
      <aside className="sidebar">
        <a className="brand" href="/">
          <span className="brand-icon">
            <Code2 size={24} />
          </span>
          DevHub<span className="version">LOCAL</span>
        </a>
        <div className="workspace-label">{t("workspace")}</div>
        <button
          className={`nav-item ${!project ? "selected" : ""}`}
          onClick={() => {
            setProject("");
            setFilter("all");
          }}
        >
          <LayoutGrid size={18} />
          {t("allServices")}
          <span>{snapshot.services.length}</span>
        </button>
        <button
          className="nav-item"
          onClick={() => {
            setProject("");
            setFilter("active");
          }}
        >
          <Activity size={18} />
          {t("runningServices")}
          <span>{running}</span>
        </button>
        <div className="workspace-label projects-label">
          {t("projects")}{" "}
          <span>{projects.length.toString().padStart(2, "0")}</span>
        </div>
        <nav aria-label={t("projects")} className="project-list">
          {projects.length ? (
            projects.map((p) => (
              <button
                key={p}
                title={p}
                className={`nav-item ${project === p ? "selected" : ""} accent-${projectPref(preferences.projectPrefs, p).color}`}
                onClick={() => setProject(p)}
              >
                <ProjectIcon
                  name={projectPref(preferences.projectPrefs, p).icon}
                  size={17}
                />
                <span className="project-name">{p}</span>
                <span>
                  {snapshot.services.filter((s) => s.project === p).length}
                </span>
              </button>
            ))
          ) : (
            <p className="sidebar-empty">{t("discoveryEmpty")}</p>
          )}
        </nav>
        <div className="sidebar-bottom">
          <div className="local-card">
            <ShieldCheck size={19} />
            <div>
              <strong>{t("localOnly")}</strong>
              <span>{t("privateWorkspace")}</span>
            </div>
          </div>
          <button className="nav-item" onClick={() => setSettings(true)}>
            <Settings2 size={18} />
            {t("settingsHelp")}
            <ChevronRight size={16} />
          </button>
          <div className="sidebar-footer">
            DEVHUB <span>v0.1.0</span>
          </div>
        </div>
      </aside>
      <main>
        <header className="topbar">
          <div>
            <span className="breadcrumb">{t("workspace")}</span>
            <ChevronRight size={14} />
            <span>{project || t("allServices")}</span>
          </div>
          <span className="connection">
            <span className={error ? "dot offline" : "dot"} />
            {error ? t("disconnected") : t("localConnection")}
          </span>
        </header>
        <div className="main-content">
          <div className="page-title">
            <div>
              <div className="eyebrow">{t("controlRoom")}</div>
              <h1>{project || t("developmentServices")}</h1>
              <p>{t("tagline")}</p>
            </div>
            <div className="title-actions">
              <button
                className="button"
                aria-label={t("switchList")}
                title={t("switchList")}
                onClick={() =>
                  setPreferences((v) => ({
                    ...v,
                    serviceView: v.serviceView === "list" ? "cards" : "list",
                  }))
                }
              >
                {preferences.serviceView === "list" ? (
                  <LayoutGrid size={16} />
                ) : (
                  <LayoutList size={16} />
                )}
              </button>
              <button
                className="button"
                disabled={!!busy || !info}
                onClick={scan}
              >
                <RefreshCw
                  size={16}
                  className={busy === "scan" ? "spin" : ""}
                />
                {t("rescan")}
              </button>
              <button
                className="button primary"
                disabled={!info}
                onClick={() => setEditor("new")}
              >
                <Plus size={17} />
                {t("addService")}
              </button>
            </div>
          </div>
          <div className="stats">
            <div>
              <span className="stat-label">
                <Server size={17} />
                {t("allServices")}
              </span>
              <strong>
                {snapshot.services.length.toString().padStart(2, "0")}
              </strong>
              <span className="stat-detail">{t("discoveredManual")}</span>
            </div>
            <div>
              <span className="stat-label">
                <Activity size={17} />
                {t("runningServices")}
              </span>
              <strong className="lime">
                {running.toString().padStart(2, "0")}
              </strong>
              <span className="stat-detail">
                {t("managedCount", {
                  count: snapshot.services.filter((s) => s.managed).length,
                })}
              </span>
            </div>
            <div>
              <span className="stat-label">
                <FolderOpen size={17} />
                {t("projects")}
              </span>
              <strong>{projects.length.toString().padStart(2, "0")}</strong>
              <span className="stat-detail">{t("byProject")}</span>
            </div>
            <div className="scan-stat">
              <span className="stat-label">
                <Radio size={17} />
                {t("autoDiscovery")}
              </span>
              <strong className="scan-value">
                {preferences.autoScan
                  ? t("scanEvery", { seconds: preferences.scanIntervalSeconds })
                  : t("scanOff")}
              </strong>
              <span className="stat-detail">
                {snapshot.lastScan
                  ? t("lastScan", {
                      time: new Date(snapshot.lastScan).toLocaleTimeString(),
                    })
                  : t("waitingScan")}
              </span>
            </div>
          </div>
          {(error || snapshot.scanError) && (
            <div role="alert" className="error-banner">
              {error
                ? t("connectError", { error })
                : t("scanError", { error: snapshot.scanError })}
            </div>
          )}
          <div className="toolbar">
            <div className="tabs" aria-label={t("statusFilter")}>
              {[
                ["all", t("all")],
                ["active", t("running")],
                ["stopped", t("stopped")],
              ].map(([id, text]) => (
                <button
                  aria-pressed={filter === id}
                  className={filter === id ? "tab active" : "tab"}
                  key={id}
                  onClick={() => setFilter(id)}
                >
                  {text}
                  {id === "all" && <span>{snapshot.services.length}</span>}
                </button>
              ))}
            </div>
            <label className="search">
              <Search size={17} />
              <input
                ref={searchRef}
                aria-label={t("searchLabel")}
                placeholder={t("searchPlaceholder")}
                value={query}
                onChange={(e) => setQuery(e.target.value)}
              />
              <kbd>/</kbd>
            </label>
          </div>
          {loading ? (
            <div className="empty">
              <Loader2 className="spin" size={30} />
              <h2>{t("connecting")}</h2>
            </div>
          ) : shown.length ? (
            <div
              className={`service-grid ${preferences.serviceView === "list" ? "list" : ""}`}
            >
              {shown.map((s) => (
                <article
                  className={`service-card accent-${projectPref(preferences.projectPrefs, s.project).color}`}
                  key={s.id}
                >
                  <div className={`cover ${s.cover ? "has-cover" : ""}`}>
                    {s.cover ? (
                      <img
                        src={`/covers/${s.cover}`}
                        alt={`${s.name} ${t("webPreview")}`}
                        loading="lazy"
                      />
                    ) : (
                      <div className="cover-placeholder">
                        <div className="terminal-mark">
                          <Terminal size={28} />
                        </div>
                        <span>{s.framework || t("localService")}</span>
                        <code>:{s.port}</code>
                      </div>
                    )}
                    <span className={`status ${s.status}`}>
                      <span />
                      {t(
                        s.status === "running"
                          ? "running"
                          : s.status === "starting"
                            ? "startup"
                            : s.status === "external"
                              ? "external"
                              : "stopped",
                      )}
                    </span>
                    <button
                      className="cover-button"
                      aria-label={`${t("updateCover")} ${s.name}`}
                      title={s.screenshotError || t("updateCover")}
                      disabled={!active(s) || !!busy}
                      onClick={() => void action(s, "capture")}
                    >
                      <Camera size={16} />
                    </button>
                  </div>
                  <div className="card-content">
                    <div className="card-heading">
                      <h2 title={s.name}>{s.name}</h2>
                      <span className="framework">
                        {s.framework || t("custom")}
                      </span>
                    </div>
                    <div className="project-caption">
                      <Folder size={13} />
                      <span>{s.project}</span>
                      <span className="origin">
                        {s.source === "discovered"
                          ? t("autoSource")
                          : t("manualSource")}
                      </span>
                    </div>
                    <div className="service-url">
                      <span className="port">:{s.port}</span>
                      <a
                        href={s.url}
                        target="_blank"
                        rel="noreferrer"
                        title={s.url}
                      >
                        {s.url.replace(/^https?:\/\//, "")}
                        <ArrowUpRight size={15} />
                      </a>
                    </div>
                    {servicePorts(s).filter((port) => port !== s.port).length >
                      0 && (
                      <p className="extra-ports">
                        {t("alsoListening", {
                          ports: servicePorts(s)
                            .filter((port) => port !== s.port)
                            .map((port) => `:${port}`)
                            .join(", "),
                        })}
                      </p>
                    )}
                    {s.source === "discovered" && (
                      <p
                        className="discovery-reason"
                        title={s.discoveryDetail || s.processName}
                      >
                        {t(
                          `discoveryReason_${s.discoveryReason || "project-marker"}`,
                        )}
                        {s.processName ? ` · ${s.processName}` : ""}
                        {s.discoveryDetail &&
                        s.discoveryReason &&
                        s.discoveryReason !== "project-marker"
                          ? ` · ${s.discoveryDetail}`
                          : ""}
                      </p>
                    )}
                    <div
                      className="health-strip"
                      aria-label={t("healthStatus")}
                    >
                      <span
                        className={`health-chip ${s.health?.process || "unknown"}`}
                        title={s.health?.error || t("processHealth")}
                      >
                        <span />
                        {t("processHealth")} ·{" "}
                        {t(
                          `health${s.health?.process === "running" ? "Running" : s.health?.process === "stopped" ? "Stopped" : "Unknown"}`,
                        )}
                      </span>
                      <span
                        className={`health-chip ${s.health?.tcp || "unknown"}`}
                        title={s.health?.error || t("tcpHealth")}
                      >
                        <span />
                        {t("tcpHealth")} ·{" "}
                        {t(
                          `health${s.health?.tcp === "reachable" ? "Reachable" : s.health?.tcp === "unreachable" ? "Unreachable" : "Unknown"}`,
                        )}
                      </span>
                      <span
                        className={`health-chip ${s.health?.http || "unknown"}`}
                        title={s.health?.error || t("httpHealth")}
                      >
                        <span />
                        {t("httpHealth")} ·{" "}
                        {t(
                          `health${s.health?.http === "healthy" ? "Healthy" : s.health?.http === "failed" ? "Failed" : s.health?.http === "unavailable" ? "Unavailable" : "Unknown"}`,
                        )}
                        {s.health?.httpStatus ? ` ${s.health.httpStatus}` : ""}
                      </span>
                    </div>
                    <div className="directory" title={s.directory}>
                      <FolderOpen size={14} />
                      <span>{s.directory}</span>
                    </div>
                    <div
                      className="command"
                      title={s.command || t("setLaunchCommand")}
                    >
                      <span>$</span>
                      <code>{s.command || t("noLaunchCommand")}</code>
                    </div>
                    {s.screenshotError && !s.cover && (
                      <p className="cover-error" title={s.screenshotError}>
                        {t("coverRetry")}
                      </p>
                    )}
                    {failureMessage(s, t) && (
                      <p className="failure-reason" role="status">
                        <span>{failureMessage(s, t)}</span>
                        {s.command &&
                          (s.failure === "start-timeout" ||
                            s.failure === "start-exit") && (
                            <button
                              className="text-button"
                              disabled={!!busy}
                              onClick={() => void retry(s)}
                            >
                              {t("retryStart")}
                            </button>
                          )}
                      </p>
                    )}
                    <div className="card-actions">
                      <button
                        className={`button ${active(s) ? "stop-button" : "start-button"}`}
                        disabled={!!busy || (!active(s) && !s.command)}
                        title={
                          !active(s) && !s.command
                            ? t("editBeforeStart")
                            : undefined
                        }
                        onClick={() =>
                          void action(s, active(s) ? "stop" : "start")
                        }
                      >
                        {busy === s.id ? (
                          <Loader2 size={15} className="spin" />
                        ) : active(s) ? (
                          <Square size={13} />
                        ) : (
                          <Play size={14} />
                        )}
                        {active(s)
                          ? t("stop")
                          : s.failure
                            ? t("retryStart")
                            : t("start")}
                      </button>
                      <button
                        className="button log-button"
                        onClick={() => setLogs(s)}
                      >
                        <Terminal size={15} />
                        {t("logs")}
                      </button>
                      <div className="card-tools">
                        <button
                          className="icon-button"
                          aria-label={`${t("editService")} ${s.name}`}
                          title={
                            active(s) ? t("editAfterStop") : t("editService")
                          }
                          disabled={active(s) || !!busy}
                          onClick={() => setEditor(s)}
                        >
                          <Pencil size={15} />
                        </button>
                        <button
                          className="icon-button"
                          aria-label={`${t("removeRecord")} ${s.name}`}
                          title={t("removeRecord")}
                          disabled={active(s) || !!busy}
                          onClick={() => void action(s, "delete")}
                        >
                          <Trash2 size={15} />
                        </button>
                      </div>
                    </div>
                  </div>
                </article>
              ))}
            </div>
          ) : (
            <div className="empty">
              <div className="empty-icon">
                <Terminal size={32} />
              </div>
              <h2>
                {snapshot.services.length ? t("noMatch") : t("emptyTitle")}
              </h2>
              <p>
                {snapshot.services.length ? t("noMatchHint") : t("emptyHint")}
              </p>
              {!snapshot.services.length && (
                <button
                  className="button primary"
                  disabled={!info}
                  onClick={() => setEditor("new")}
                >
                  <Plus size={17} />
                  {t("addFirst")}
                </button>
              )}
              <div className="empty-note">
                <Radio size={16} />
                {t("scanNote")}
              </div>
            </div>
          )}
          <footer className="content-footer">
            <span>
              <ShieldCheck size={14} />
              {t("savedLocal")}
            </span>
            <button onClick={() => setSettings(true)}>
              <CircleHelp size={14} />
              {t("notFound")}
            </button>
          </footer>
        </div>
      </main>
      {notice && (
        <div className="toast" role="status">
          {notice}
          <button
            className="icon-button"
            aria-label={t("close")}
            onClick={() => setNotice("")}
          >
            <X size={16} />
          </button>
        </div>
      )}
      {editor && (
        <Editor
          service={editor === "new" ? undefined : editor}
          close={() => setEditor(undefined)}
          saved={() => {
            void refresh();
            setNotice(t("serviceSaved"));
          }}
          platform={info?.platform}
          language={language}
        />
      )}
      {logs && (
        <Logs
          service={snapshot.services.find((s) => s.id === logs.id) || logs}
          close={() => setLogs(undefined)}
          language={language}
        />
      )}
      {confirm && (
        <Modal
          title={
            confirm.action === "delete"
              ? t("removeTitle")
              : t("stopExternalTitle")
          }
          close={() => setConfirm(undefined)}
          closeLabel={t("close")}
        >
          <p>
            {confirm.action === "delete"
              ? t("removeConfirm", { name: confirm.service.name })
              : t("stopExternalConfirm", {
                  name: confirm.service.name,
                  pid: confirm.service.pid,
                })}
          </p>
          <div className="modal-actions">
            <button className="button" onClick={() => setConfirm(undefined)}>
              {t("cancel")}
            </button>
            <button
              className="button danger"
              onClick={() => void confirmAction()}
            >
              {confirm.action === "delete" ? t("remove") : t("stopProcess")}
            </button>
          </div>
        </Modal>
      )}
      {settings && (
        <SettingsModal
          settings={preferences}
          info={info}
          language={language}
          projects={projects}
          close={() => setSettings(false)}
          saved={(value) => {
            setPreferences(value);
            setSettings(false);
            void refresh();
            setNotice(createTranslator(value.language)("settingsSaved"));
          }}
        />
      )}
    </div>
  );
}
