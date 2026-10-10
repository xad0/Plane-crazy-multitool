import { useEffect, useMemo, useRef, useState } from "react";
import { calculateCrank, calculateDyno, calculateDynoCurve, dynoPlotCoordinates, formatDynoExportFilename, calculateGear, calculatePiston, calculateWheel, DynoPoint, DynoGraphMode, round3, CrankLayout } from "./lib/math";

type Tool = "gear" | "crank" | "wheel" | "dyno" | "piston" | "settings";
type Theme = "catppuccin-mocha" | "catppuccin-latte" | "catppuccin-frappe" | "catppuccin-macchiato" | "nord" | "gruvbox" | "solid";
type Side = "left" | "right";
type Settings = {
  theme: Theme; accent: string; referenceMin: number; referenceMax: number; referenceEnabled: boolean;
  columns: Record<string, boolean>; calculator: Record<string, boolean>;
  gearLayout: "automatic" | "calculator-left" | "calculator-right" | "calculator-only" | "reference-only" | "stacked";
  crankSide: Side; wheelSide: Side; dynoSide: Side; pistonSide: Side;
};
const storageKey = "pc-multitool.web.v1";
const defaults: Settings = { theme: "catppuccin-mocha", accent: "green", referenceMin: 4, referenceMax: 20, referenceEnabled: true, gearLayout: "automatic", crankSide: "right", wheelSide: "right", dynoSide: "right", pistonSide: "right", columns: { teeth: true, full: true, half: true, offset: true, value: true, compressors: true }, calculator: { teeth: true, compressors: true, full: true, half: true, offset: true, value: true, warnings: true } };
const tools: Array<[Tool, string, string]> = [["gear", "Gear", "Gear calculator"], ["crank", "Crank", "Crank angle calculator"], ["wheel", "Wheel", "Wheel calculator"], ["dyno", "Dyno", "Power graph"], ["piston", "Piston", "Piston length"], ["settings", "Settings", "Appearance and reference"]];
const fmt = (value: number, suffix = "") => `${value.toFixed(3)}${suffix}`;
const palette: Record<Theme, Record<string, string>> = {
  "catppuccin-latte": { "--bg":"#eff1f5", "--surface":"#e6e9ef", "--panel":"#ccd0da", "--text":"#4c4f69", "--muted":"#6c6f85", "--border":"#9ca0b0" },
  "catppuccin-frappe": { "--bg":"#303446", "--surface":"#292c3c", "--panel":"#414559", "--text":"#c6d0f5", "--muted":"#a5adce", "--border":"#737994" },
  "catppuccin-macchiato": { "--bg":"#24273a", "--surface":"#1e2030", "--panel":"#363a4f", "--text":"#cad3f5", "--muted":"#a5adcb", "--border":"#6e738d" },
  "catppuccin-mocha": { "--bg":"#11111b", "--surface":"#181825", "--panel":"#1e1e2e", "--text":"#cdd6f4", "--muted":"#7f849c", "--border":"#45475a" },
  nord: { "--bg":"#2e3440", "--surface":"#3b4252", "--panel":"#434c5e", "--text":"#eceff4", "--muted":"#88c0d0", "--border":"#4c566a" },
  gruvbox: { "--bg":"#1d2021", "--surface":"#282828", "--panel":"#3c3836", "--text":"#ebdbb2", "--muted":"#a89984", "--border":"#504945" },
  solid: { "--bg":"#080808", "--surface":"#101010", "--panel":"#151515", "--text":"#eeeeee", "--muted":"#777777", "--border":"#303030" },
};

function useStoredSettings() {
  const [settings, setSettings] = useState<Settings>(() => {
    try { const saved = JSON.parse(localStorage.getItem(storageKey) ?? "{}"); return { ...defaults, ...saved, columns: { ...defaults.columns, ...saved.columns }, calculator: { ...defaults.calculator, ...saved.calculator } }; }
    catch { return defaults; }
  });
  useEffect(() => localStorage.setItem(storageKey, JSON.stringify(settings)), [settings]);
  return [settings, setSettings] as const;
}

// ── Copy-to-clipboard shared mechanism ────────────────────────────────────────
// Singleton toast bus so all CopyValue instances share one overlay.
type Toast = { id: number; message: string };
let _toastListeners: Array<(toasts: Toast[]) => void> = [];
let _toasts: Toast[] = [];
let _toastCounter = 0;

function _addToast(message: string) {
  const id = ++_toastCounter;
  _toasts = [..._toasts, { id, message }];
  _toastListeners.forEach(fn => fn(_toasts));
  setTimeout(() => {
    _toasts = _toasts.filter(t => t.id !== id);
    _toastListeners.forEach(fn => fn(_toasts));
  }, 2200);
}

async function _copyToClipboard(rawValue: string) {
  try {
    if (navigator.clipboard && typeof navigator.clipboard.writeText === "function") {
      await navigator.clipboard.writeText(rawValue);
      _addToast(`Copied ${rawValue}`);
      return;
    }
  } catch {
    // Try the legacy API in browsers or contexts without Clipboard API access.
  }

  const el = document.createElement("textarea");
  el.value = rawValue;
  el.setAttribute("readonly", "");
  el.style.cssText = "position:fixed;opacity:0;pointer-events:none";
  let copied = false;
  try {
    document.body.appendChild(el);
    el.select();
    copied = document.execCommand("copy");
  } catch {
    copied = false;
  } finally {
    el.remove();
  }
  _addToast(copied ? `Copied ${rawValue}` : "Copy failed: clipboard access was denied");
}

// Mounted once at the root; subscribes to the singleton bus.
function CopyToastContainer() {
  const [toasts, setToasts] = useState<Toast[]>(_toasts);
  useEffect(() => {
    _toastListeners.push(setToasts);
    return () => { _toastListeners = _toastListeners.filter(fn => fn !== setToasts); };
  }, []);
  if (!toasts.length) return null;
  return (
    <div className="copy-toasts" aria-live="polite" aria-atomic="false">
      {toasts.map(t => <div key={t.id} className="copy-toast">{t.message}</div>)}
    </div>
  );
}

// Wraps a displayed value. Hovering highlights it; clicking copies `rawValue`.
// `children` = displayed text (may include units); `rawValue` = bare number string.
function CopyValue({ rawValue, children }: { rawValue: string; children: React.ReactNode }) {
  return (
    <span
      className="copy-value"
      title={`Click to copy ${rawValue}`}
      onClick={() => _copyToClipboard(rawValue)}
      role="button"
      tabIndex={0}
      onKeyDown={e => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); _copyToClipboard(rawValue); } }}
    >
      {children}
    </span>
  );
}

// ── Shared UI primitives ───────────────────────────────────────────────────────

function Card({ title, children, className = "" }: { title: string; children: React.ReactNode; className?: string }) {
  return <section className={`card ${className}`}><h2>{title}</h2>{children}</section>;
}

function Field({ label, value, onChange, min, step = "1", hint }: { label: string; value: string; onChange: (value: string) => void; min?: number; step?: string; hint?: string }) {
  return <label className="field"><span>{label}</span><input type="number" inputMode="decimal" value={value} min={min} step={step} onChange={(e) => onChange(e.target.value)} />{hint && <small>{hint}</small>}</label>;
}

// Result: displays a label + value pair. When rawValue is provided that is what
// goes to the clipboard (stripped of units); otherwise value itself is copied.
// Pass rawValue={null} to make a result non-copyable (e.g. "N/A").
function Result({ label, value, rawValue }: { label: string; value: string; rawValue?: string | null }) {
  const raw = rawValue === null ? undefined : (rawValue ?? value);
  return (
    <div className="result">
      <span>{label}</span>
      <strong>
        {raw !== undefined
          ? <CopyValue rawValue={raw}>{value}</CopyValue>
          : value}
      </strong>
    </div>
  );
}

// ── Gear calculator ────────────────────────────────────────────────────────────

function GearTool({ settings }: { settings: Settings }) {
  const [teeth, setTeeth] = useState("6"), [compressors, setCompressors] = useState("1"), [manual, setManual] = useState(false);
  const toothValue = Number(teeth), result = calculateGear(toothValue, manual && settings.calculator.compressors ? Number(compressors) : undefined);
  const gearLayout = !settings.referenceEnabled && settings.gearLayout === "reference-only" ? "calculator-only" : settings.gearLayout;
  useEffect(() => { if (result && !manual) setCompressors(String(result.compressors)); }, [teeth, manual, result?.compressors]);
  const reference = useMemo(() => Array.from({ length: Math.min(200, Math.max(0, settings.referenceMax - settings.referenceMin + 1)) }, (_, i) => calculateGear(settings.referenceMin + i)!), [settings.referenceMin, settings.referenceMax]);
  return (
    <ToolLayout title="Gear calculator" intro="" className={`gear-layout gear-layout-${gearLayout}${settings.referenceEnabled ? "" : " gear-no-reference"}`}>
      <div className="gear-calculator">
      <Card title="Inputs" className="tool-inputs">
        <Field label="Number of teeth" value={teeth} onChange={(v) => { setTeeth(v); setManual(false); }} min={3} />
        {settings.calculator.compressors && <>
          <div className="switch">
            <input aria-label="Automatic compressors" type="checkbox" checked={!manual} onChange={(e) => setManual(!e.target.checked)} />
            <span>Automatic compressors</span>
          </div>
          <Field label="Compressors" value={compressors} onChange={(v) => { setCompressors(v); setManual(true); }} min={1} />
        </>}
      </Card>
      <Card title="Result" className="tool-results">
        {!result
          ? <Notice tone="error">Enter a whole tooth count of 3 or more. Teeth 1–2 are mathematically invalid.</Notice>
          : <>
            <div className="results-grid">
              {settings.calculator.teeth && <Result label="Teeth" value={String(toothValue)} />}
              {settings.calculator.compressors && <Result label="Compressors" value={String(result.compressors)} />}
              {settings.calculator.full && <Result label="Full angle" value={fmt(result.full, "°")} rawValue={result.full.toFixed(3)} />}
              {settings.calculator.half && <Result label="Half angle" value={fmt(result.half, "°")} rawValue={result.half.toFixed(3)} />}
              {settings.calculator.offset && <Result label="Offset" value={fmt(result.offset)} rawValue={result.offset.toFixed(3)} />}
              {settings.calculator.value && <Result label="Compressor value" value={fmt(round3(result.compressorValue))} rawValue={round3(result.compressorValue).toFixed(3)} />}
            </div>
            {!Object.entries(settings.calculator).some(([key, enabled]) => key !== "warnings" && enabled) && <p className="muted">Enable result fields in Settings → Calculator.</p>}
            {settings.calculator.warnings && result.compressorValue > 1 && <Notice tone="warn">MORE COMPRESSORS NEEDED</Notice>}
          </>}
      </Card>
      {settings.referenceEnabled && <Card title={`Gear reference · ${settings.referenceMin}–${settings.referenceMax} teeth`} className="gear-reference wide">
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                {settings.columns.teeth       && <th>Teeth</th>}
                {settings.columns.full        && <th>Full angle</th>}
                {settings.columns.half        && <th>Half angle</th>}
                {settings.columns.offset      && <th>Offset</th>}
                {settings.columns.value       && <th>Comp. value</th>}
                {settings.columns.compressors && <th>Compressors</th>}
              </tr>
            </thead>
            <tbody>
              {reference.map((row, i) => (
                <tr key={settings.referenceMin + i}>
                  {settings.columns.teeth       && <td>{settings.referenceMin + i}</td>}
                  {settings.columns.full        && <td><CopyValue rawValue={row.full.toFixed(3)}>{fmt(row.full, "°")}</CopyValue></td>}
                  {settings.columns.half        && <td><CopyValue rawValue={row.half.toFixed(3)}>{fmt(row.half, "°")}</CopyValue></td>}
                  {settings.columns.offset      && <td><CopyValue rawValue={row.offset.toFixed(3)}>{fmt(row.offset)}</CopyValue></td>}
                  {settings.columns.value       && <td><CopyValue rawValue={round3(row.compressorValue).toFixed(3)}>{fmt(round3(row.compressorValue))}</CopyValue></td>}
                  {settings.columns.compressors && <td><CopyValue rawValue={String(row.compressors)}>{row.compressors}</CopyValue></td>}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>}
      </div>
    </ToolLayout>
  );
}

// ── Crank angle calculator ─────────────────────────────────────────────────────

function CrankTool({ settings }: { settings: Settings }) {
  const [layout, setLayout] = useState<CrankLayout>("Inline"), [cylinders, setCylinders] = useState("6");
  const result = calculateCrank(layout, Number(cylinders));
  return (
    <ToolLayout title="Crank angle" intro="Find phase spacing for inline, V, and boxer engines." className={settings.crankSide === "left" ? "side-left" : "side-right"}>
      <Card title="Inputs" className="tool-inputs">
        <label className="field">
          <span>Layout</span>
          <select value={layout} onChange={(e) => setLayout(e.target.value as CrankLayout)}>
            {(["Inline", "V", "Boxer"] as const).map(x => <option key={x}>{x}</option>)}
          </select>
        </label>
        <Field label="Cylinders" value={cylinders} onChange={setCylinders} min={1} />
      </Card>
      <Card title="Phase positions" className="tool-results">
        {!result
          ? <Notice tone="error">Cylinder count must be positive.</Notice>
          : <>
            <div className="results-grid">
              <Result label="Effective positions" value={String(result.positions)} />
              <Result label="Base phase spacing"  value={fmt(result.angle, "°")} rawValue={result.angle.toFixed(3)} />
            </div>
            <ol className="phase-list">
              {result.phases.map((phase, i) => (
                <li key={i}>
                  <span>Position {i + 1}</span>
                  <strong><CopyValue rawValue={phase.toFixed(3)}>{fmt(phase, "°")}</CopyValue></strong>
                </li>
              ))}
            </ol>
          </>}
      </Card>
    </ToolLayout>
  );
}

// ── Wheel calculator ───────────────────────────────────────────────────────────

function WheelTool({ settings }: { settings: Settings }) {
  const [size, setSize] = useState("10");
  const result = calculateWheel(Number(size));
  return (
    <ToolLayout title="Wheel calculator" intro="Wheel lock angles and compressor values use their own calculation rules." className={settings.wheelSide === "left" ? "side-left" : "side-right"}>
      <Card title="Input" className="tool-inputs">
        <Field label="Tire size (sm)" value={size} onChange={setSize} min={1} />
      </Card>
      <Card title="Tire result" className="tool-results">
        {!result
          ? <Notice tone="error">Enter a positive whole tire size.</Notice>
          : <div className="results-grid">
            <Result label="Angle lock"        value={fmt(result.angleLock, "°")}   rawValue={result.angleLock.toFixed(3)} />
            <Result label="Center angle"      value={result.centerAngle === null ? "N/A" : fmt(result.centerAngle, "°")} rawValue={result.centerAngle === null ? null : result.centerAngle.toFixed(3)} />
            <Result label="Offset"            value={fmt(round3(result.offset))}   rawValue={round3(result.offset).toFixed(3)} />
            <Result label="Compressors"       value={String(result.compressors)} />
            <Result label="Compressor value"  value={fmt(result.compressorValue)}  rawValue={result.compressorValue.toFixed(3)} />
          </div>}
      </Card>
    </ToolLayout>
  );
}

// ── Piston length calculator ───────────────────────────────────────────────────

function PistonTool({ settings }: { settings: Settings }) {
  const [distance, setDistance] = useState("1"), [pistons, setPistons] = useState("2"), [value, setValue] = useState("3.106");
  const length = calculatePiston(Number(distance), Number(pistons), Number(value));
  return (
    <ToolLayout title="Piston length" intro="Calculate piston length from crank distance, piston count, and value." className={settings.pistonSide === "left" ? "side-left" : "side-right"}>
      <Card title="Inputs" className="tool-inputs">
        <Field label="Crank distance"    value={distance} onChange={setDistance} min={0} step="any" />
        <Field label="Amount of pistons" value={pistons}  onChange={setPistons}  min={1} />
        <Field label="Value"             value={value}    onChange={setValue}    min={0} step="any" />
      </Card>
      <Card title="Piston length" className="tool-results">
        {length === null
          ? <Notice tone="error">Enter valid non-negative values and at least one piston.</Notice>
          : <>
            <Result label="Calculated length" value={fmt(length)} rawValue={length.toFixed(3)} />
            {length > 4 && <Notice tone="warn">VALUE EXCEEDS LIMIT</Notice>}
          </>}
      </Card>
    </ToolLayout>
  );
}

// ── Dyno tool ─────────────────────────────────────────────────────────────────

type DynoRow = { sps: string; torque: string };
function DynoTool({ settings }: { settings: Settings }) {
  const [rows, setRows] = useState<DynoRow[]>(Array.from({ length: 5 }, () => ({ sps: "", torque: "" })));
  const [graphMode, setGraphMode] = useState<DynoGraphMode>("bhp-rpm");
  const points = calculateDyno(rows);
  const measurements = points.slice(1);
  const update = (index: number, key: keyof DynoRow, value: string) =>
    setRows(rows.map((r, i) => i === index ? { ...r, [key]: value } : r));
  const removeRow = (index: number) => setRows(current => current.length > 1 ? current.filter((_, i) => i !== index) : current);
  const peaks = measurements.length
    ? {
        torque: measurements.reduce((a, b) => b.torque > a.torque ? b : a),
        sps:    measurements.reduce((a, b) => b.sps    > a.sps    ? b : a),
        bhp:    measurements.reduce((a, b) => b.bhp    > a.bhp    ? b : a),
      }
    : null;
  return (
    <ToolLayout title="Dyno" intro="Enter SPS and torque samples, then choose BHP/RPM or torque/SPS." className={settings.dynoSide === "left" ? "side-left" : "side-right"}>
      <Card title="Dyno data" className="dyno-data tool-inputs">
        <div className="dyno-table">
          <div className="dyno-head"><span>#</span><span>SPS</span><span>Torque</span><span aria-hidden="true" /></div>
          {rows.map((row, i) => (
            <div className="dyno-row" key={i}>
              <span>{i + 1}</span>
              <input aria-label={`SPS row ${i + 1}`}    inputMode="decimal" value={row.sps}    onChange={(e) => update(i, "sps",    e.target.value)} />
              <input aria-label={`Torque row ${i + 1}`} inputMode="decimal" value={row.torque} onChange={(e) => update(i, "torque", e.target.value)} />
              <button type="button" className="row-remove" aria-label={`Remove dyno row ${i + 1}`} title="Remove row" disabled={rows.length <= 1} onClick={() => removeRow(i)}>×</button>
            </div>
          ))}
        </div>
        <button className="secondary" onClick={() => setRows([...rows, { sps: "", torque: "" }])}>Add row</button>
      </Card>
      <Card title={graphMode === "bhp-rpm" ? "BHP vs RPM" : "Torque vs SPS"} className="dyno-chart tool-results">
        <div className="graph-mode-switch" role="group" aria-label="Dyno graph mode">
          <button type="button" className={graphMode === "bhp-rpm" ? "active" : ""} aria-pressed={graphMode === "bhp-rpm"} onClick={() => setGraphMode("bhp-rpm")}>BHP vs RPM</button>
          <button type="button" className={graphMode === "torque-sps" ? "active" : ""} aria-pressed={graphMode === "torque-sps"} onClick={() => setGraphMode("torque-sps")}>Torque vs SPS</button>
        </div>
        <DynoChart points={points} mode={graphMode} />
        {peaks
          ? <div className="peaks">
              <Result label="Peak torque" value={`${fmt(peaks.torque.torque)} @ ${fmt(peaks.torque.sps)} SPS`}    rawValue={peaks.torque.torque.toFixed(3)} />
              <Result label="Peak SPS"    value={`${fmt(peaks.sps.sps)} @ ${fmt(peaks.sps.torque)} torque`}        rawValue={peaks.sps.sps.toFixed(3)} />
              <Result label="Peak BHP"    value={`${fmt(peaks.bhp.bhp)} @ ${fmt(peaks.bhp.sps)} SPS`}             rawValue={peaks.bhp.bhp.toFixed(3)} />
            </div>
          : <p className="muted">Enter complete SPS and torque rows to begin.</p>}
      </Card>
    </ToolLayout>
  );
}

// Keep same-second exports unique within this page session. If the browser finds
// a prior file with the same name in Downloads, it can still apply its own suffix.
const dynoExportCounts = new Map<string, number>();
function nextDynoExportFilename(date: Date) {
  const base = formatDynoExportFilename(date);
  const sequence = dynoExportCounts.get(base) ?? 0;
  dynoExportCounts.set(base, sequence + 1);
  return formatDynoExportFilename(date, sequence);
}

type DynoDrawOptions = { width?: number; height?: number; pixelRatio?: number; standalone?: boolean };

function DynoChart({ points, mode }: { points: DynoPoint[]; mode: DynoGraphMode }) {
  const canvas = useRef<HTMLCanvasElement>(null), chartWrap = useRef<HTMLDivElement>(null), [hover, setHover] = useState<DynoPoint | null>(null), [fullscreen, setFullscreen] = useState(false);
  useEffect(() => {
    const el = canvas.current; if (!el) return;
    const draw = () => drawDyno(el, points, mode);
    const observer = new ResizeObserver(draw); observer.observe(el); draw(); return () => observer.disconnect();
  }, [points, mode]);
  useEffect(() => {
    const sync = () => setFullscreen(document.fullscreenElement === chartWrap.current);
    document.addEventListener("fullscreenchange", sync);
    sync();
    return () => document.removeEventListener("fullscreenchange", sync);
  }, []);
  const pointer = (event: React.PointerEvent<HTMLCanvasElement>) => {
    const rect = event.currentTarget.getBoundingClientRect();
    if (points.length <= 1) return;
    const left = 58, right = 18;
    const ratio = Math.max(0, Math.min(1, (event.clientX - rect.left - left) / Math.max(1, rect.width - left - right)));
    const value = (point: DynoPoint) => mode === "torque-sps" ? point.sps : point.rpm;
    const min = value(points[0]), max = value(points[points.length - 1]);
    const target = min + ratio * (max - min);
    setHover(points.reduce((nearest, point) => Math.abs(value(point) - target) < Math.abs(value(nearest) - target) ? point : nearest));
  };
  const download = () => {
    if (points.length <= 1) return;
    const image = document.createElement("canvas");
    drawDyno(image, points, mode, { width: 1600, height: 900, pixelRatio: 1, standalone: true });
    const link = document.createElement("a");
    link.download = nextDynoExportFilename(new Date());
    link.href = image.toDataURL("image/png");
    link.click();
  };
  const toggleFullscreen = async () => { const wrap = chartWrap.current; if (!wrap) return; try { if (document.fullscreenElement === wrap) await document.exitFullscreen(); else await wrap.requestFullscreen(); } catch { setFullscreen(false); } };
  return (
    <div ref={chartWrap} className="chart-wrap">
      <canvas ref={canvas} className="chart" role="img" aria-label={mode === "bhp-rpm" ? "Smooth BHP versus RPM Dyno graph" : "Smooth torque versus SPS Dyno graph"} onPointerMove={pointer} onPointerLeave={() => setHover(null)} />
      {hover && <output className="chart-tip">{mode === "torque-sps" ? `${fmt(hover.sps)} SPS · ${fmt(hover.torque)} torque` : `${fmt(hover.rpm)} RPM · ${fmt(hover.bhp)} BHP`}</output>}
      <div className="chart-actions">
        <button className="secondary" onClick={toggleFullscreen}>{fullscreen ? "Exit fullscreen" : "Fullscreen"}</button>
        <button className="secondary" disabled={points.length <= 1} onClick={download}>Save PNG</button>
      </div>
    </div>
  );
}

function drawDyno(canvas: HTMLCanvasElement, points: DynoPoint[], mode: DynoGraphMode, options: DynoDrawOptions = {}) {
  const rect = canvas.getBoundingClientRect();
  const dpr = options.pixelRatio ?? Math.min(window.devicePixelRatio || 1, 2);
  const w = Math.max(280, options.width ?? rect.width), h = Math.max(260, options.height ?? rect.height);
  canvas.width = Math.round(w * dpr); canvas.height = Math.round(h * dpr);
  const ctx = canvas.getContext("2d"); if (!ctx) return;
  ctx.scale(dpr, dpr); ctx.clearRect(0, 0, w, h);
  const css = getComputedStyle(document.documentElement);
  const color = (name: string) => css.getPropertyValue(name).trim();
  const surface = color("--surface"), muted = color("--muted"), border = color("--border"), accent = color("--accent"), text = color("--text");
  ctx.fillStyle = surface; ctx.fillRect(0, 0, w, h);

  const standalone = options.standalone ?? false;
  const left = standalone ? 175 : 58, right = standalone ? 110 : 18;
  const top = standalone ? 125 : 18, bottom = standalone ? 170 : 44;
  const pw = Math.max(1, w - left - right), ph = Math.max(1, h - top - bottom);
  ctx.font = standalone ? "18px system-ui" : "12px system-ui";
  ctx.strokeStyle = border; ctx.fillStyle = muted; ctx.lineWidth = 1;
  if (points.length <= 1) {
    ctx.textAlign = "center"; ctx.textBaseline = "middle";
    ctx.fillText("Enter SPS and torque data", w / 2, h / 2);
    return;
  }

  const mapped = dynoPlotCoordinates(points, mode);
  let minX = mapped[0].x, maxX = mapped[0].x, minY = mapped[0].y, maxY = mapped[0].y;
  for (const point of mapped) {
    minX = Math.min(minX, point.x); maxX = Math.max(maxX, point.x);
    minY = Math.min(minY, point.y); maxY = Math.max(maxY, point.y);
  }
  minY = Math.min(0, minY);
  if (maxX <= minX) maxX = minX + 100;
  if (maxY <= minY) maxY = minY + 100;
  const x = (value: number) => left + (value - minX) / (maxX - minX) * pw;
  const y = (value: number) => top + (1 - (value - minY) / (maxY - minY)) * ph;
  const xLabel = mode === "torque-sps" ? "SPS" : "RPM";
  const yLabel = mode === "torque-sps" ? "Torque" : "BHP";

  if (standalone) {
    ctx.textAlign = "left"; ctx.textBaseline = "alphabetic"; ctx.fillStyle = text; ctx.font = "bold 28px system-ui";
    ctx.fillText(mode === "torque-sps" ? "TORQUE VS SPS" : "BHP VS RPM", left, 38);
    ctx.fillStyle = muted; ctx.font = "20px system-ui"; ctx.fillText(yLabel.toUpperCase(), left, 82);
  }

  // Clean plotting area: axes and short external ticks only, no interior grid.
  const axisBottom = top + ph, axisRight = left + pw;
  ctx.strokeStyle = border; ctx.lineWidth = 1; ctx.beginPath();
  ctx.moveTo(left, top); ctx.lineTo(left, axisBottom);
  ctx.moveTo(axisRight, top); ctx.lineTo(axisRight, axisBottom);
  ctx.moveTo(left, axisBottom); ctx.lineTo(axisRight, axisBottom);
  ctx.stroke();

  for (let i = 0; i <= 5; i++) {
    const ratio = i / 5;
    const gy = top + ph * ratio, gx = left + pw * ratio;
    const yValue = maxY - (maxY - minY) * ratio;
    const xValue = minX + (maxX - minX) * ratio;
    ctx.beginPath();
    ctx.moveTo(left - 8, gy); ctx.lineTo(left, gy);
    ctx.moveTo(axisRight, gy); ctx.lineTo(axisRight + 8, gy);
    ctx.moveTo(gx, axisBottom); ctx.lineTo(gx, axisBottom + 8);
    ctx.stroke();
    ctx.fillStyle = muted; ctx.font = standalone ? "18px system-ui" : "12px system-ui";
    ctx.textAlign = "right"; ctx.textBaseline = "alphabetic";
    ctx.fillText(yValue.toFixed(1), left - 13, gy + 4);
    ctx.textAlign = "center";
    ctx.fillText(xValue.toFixed(0), gx, axisBottom + (standalone ? 24 : 20));
  }

  if (standalone) {
    ctx.fillStyle = muted; ctx.font = "20px system-ui"; ctx.textAlign = "center";
    ctx.fillText(xLabel.toUpperCase(), left + pw / 2, axisBottom + 58);
  } else {
    ctx.fillStyle = muted; ctx.font = "12px system-ui"; ctx.textAlign = "center";
    ctx.fillText(xLabel, left + pw / 2, h - 2);
    ctx.save(); ctx.translate(13, top + ph / 2); ctx.rotate(-Math.PI / 2);
    ctx.fillText(yLabel, 0, 0); ctx.restore();
  }

  const curve = calculateDynoCurve(points, mode, pw * dpr, ph * dpr);
  ctx.strokeStyle = accent; ctx.lineWidth = standalone ? 4 : 3; ctx.lineJoin = "round"; ctx.lineCap = "round"; ctx.beginPath();
  ctx.moveTo(x(curve[0].x), y(curve[0].y));
  for (let i = 1; i < curve.length; i++) ctx.lineTo(x(curve[i].x), y(curve[i].y));
  ctx.stroke();
  ctx.fillStyle = accent;
  mapped.forEach(point => { ctx.beginPath(); ctx.arc(x(point.x), y(point.y), standalone ? 4.5 : 3.5, 0, Math.PI * 2); ctx.fill(); });
}

// ── Settings tool ─────────────────────────────────────────────────────────────

function SettingsTool({ settings, setSettings }: { settings: Settings; setSettings: React.Dispatch<React.SetStateAction<Settings>> }) {
  const [section, setSection] = useState<"appearance" | "layout" | "reference" | "calculator">("appearance");
  const columns: Array<[string, string]>    = [["teeth", "Teeth"], ["full", "Full angle"], ["half", "Half angle"], ["offset", "Offset"], ["value", "Compressor value"], ["compressors", "Compressors"]];
  const calculator: Array<[string, string]> = [["teeth", "Number of teeth"], ["compressors", "Compressors"], ["full", "Full angle"], ["half", "Half angle"], ["offset", "Offset"], ["value", "Compressor value"], ["warnings", "Warnings"]];
  const toggle = (group: "columns" | "calculator", key: string) => setSettings(s => ({ ...s, [group]: { ...s[group], [key]: !s[group][key] } }));
  return (
    <ToolLayout title="Settings" intro="The same organization as the desktop app, adapted for browser controls.">
      <div className="settings-tabs" role="tablist">
        {([ ["appearance", "Appearance"], ["layout", "Layout"], ["reference", "Reference"], ["calculator", "Calculator"] ] as const).map(([key, label]) => (
          <button key={key} role="tab" aria-selected={section === key} className={section === key ? "active" : ""} onClick={() => setSection(key)}>{label}</button>
        ))}
      </div>
      {section === "appearance" && (
        <Card title="Theme">
          <label className="field">
            <span>Theme family</span>
            <select value={settings.theme.startsWith("catppuccin") ? "catppuccin" : settings.theme} onChange={(e) => { const theme = e.target.value === "catppuccin" ? "catppuccin-mocha" : e.target.value as Theme; setSettings(s => ({ ...s, theme })); }}>
              <option value="catppuccin">Catppuccin</option><option value="nord">Nord</option><option value="gruvbox">Gruvbox</option><option value="solid">Solid</option>
            </select>
          </label>
          {settings.theme.startsWith("catppuccin") && (
            <label className="field">
              <span>Catppuccin flavor</span>
              <select value={settings.theme} onChange={(e) => setSettings(s => ({ ...s, theme: e.target.value as Theme }))}>
                <option value="catppuccin-latte">Latte</option><option value="catppuccin-frappe">Frappé</option><option value="catppuccin-macchiato">Macchiato</option><option value="catppuccin-mocha">Mocha</option>
              </select>
            </label>
          )}
          <label className="field">
            <span>Accent</span>
            <select value={settings.accent} onChange={(e) => setSettings(s => ({ ...s, accent: e.target.value }))}>
              {["green", "blue", "purple", "red", "orange", "cyan", "pink"].map(a => <option key={a}>{a}</option>)}
            </select>
          </label>
        </Card>
      )}
      {section === "layout" && (
        <>
          <Card title="Gear calculator layout">
            <label className="field">
              <span>Desktop arrangement</span>
              <select value={settings.gearLayout} onChange={(e) => setSettings(s => ({ ...s, gearLayout: e.target.value as Settings["gearLayout"] }))}>
                {[["automatic", "Automatic"], ["calculator-left", "Calculator left"], ["calculator-right", "Calculator right"], ["calculator-only", "Calculator only"], ["reference-only", "Reference only"], ["stacked", "Stacked"]].map(([v, l]) => <option value={v} key={v}>{l}</option>)}
              </select>
            </label>
          </Card>
          <Card title="Other tools">
            <SideSelect label="Crank results"  value={settings.crankSide}  change={(v) => setSettings(s => ({ ...s, crankSide: v }))} />
            <SideSelect label="Wheel results"  value={settings.wheelSide}  change={(v) => setSettings(s => ({ ...s, wheelSide: v }))} />
            <SideSelect label="Dyno graph"     value={settings.dynoSide}   change={(v) => setSettings(s => ({ ...s, dynoSide: v }))} />
            <SideSelect label="Piston results" value={settings.pistonSide} change={(v) => setSettings(s => ({ ...s, pistonSide: v }))} />
          </Card>
        </>
      )}
      {section === "reference" && (
        <>
          <Card title="Reference chart">
            <label className="switch">
              <input type="checkbox" checked={settings.referenceEnabled} onChange={() => setSettings(s => ({ ...s, referenceEnabled: !s.referenceEnabled }))} />
              <span>Show reference chart</span>
            </label>
            <Field label="Smallest gear" value={String(settings.referenceMin)} onChange={(v) => setSettings(s => ({ ...s, referenceMin: Math.max(3, Number(v) || 3) }))} min={3} />
            <Field label="Maximum gear"  value={String(settings.referenceMax)} onChange={(v) => setSettings(s => ({ ...s, referenceMax: Math.max(s.referenceMin, Math.min(202, Number(v) || s.referenceMin)) }))} min={settings.referenceMin} />
          </Card>
          <Card title="Reference columns">
            <div className="checks">
              {columns.map(([key, label]) => <label key={key}><input type="checkbox" checked={settings.columns[key]} onChange={() => toggle("columns", key)} /> {label}</label>)}
            </div>
          </Card>
        </>
      )}
      {section === "calculator" && (
        <Card title="Gear calculator fields">
          <div className="checks">
            {calculator.map(([key, label]) => <label key={key}><input type="checkbox" checked={settings.calculator[key]} onChange={() => toggle("calculator", key)} /> {label}</label>)}
          </div>
        </Card>
      )}
    </ToolLayout>
  );
}

function SideSelect({ label, value, change }: { label: string; value: Side; change: (value: Side) => void }) {
  return <label className="field"><span>{label}</span><select value={value} onChange={(e) => change(e.target.value as Side)}><option value="right">Right</option><option value="left">Left</option></select></label>;
}

function ToolLayout({ title, children, className = "" }: { title: string; intro?: string; children: React.ReactNode; className?: string }) {
  return <main><header className="page-header"><h1>{title}</h1></header><div className={`tool-grid ${className}`.trim()}>{children}</div></main>;
}

function Notice({ tone, children }: { tone: "warn" | "error"; children: React.ReactNode }) {
  return <p className={`notice ${tone}`}>{children}</p>;
}

export function App() {
  const [settings, setSettings] = useStoredSettings(), [tool, setTool] = useState<Tool>("gear");
  const theme = settings.theme in palette ? settings.theme : "catppuccin-mocha";
  useEffect(() => {
    const root = document.documentElement;
    root.dataset.theme = theme;
    root.dataset.accent = settings.accent;
    Object.entries(palette[theme]).forEach(([name, value]) => root.style.setProperty(name, value));
    document.querySelector('meta[name="theme-color"]')?.setAttribute("content", palette[theme]["--bg"]);
  }, [theme, settings.accent]);
  return (
    <div className="app-shell">
      <header className="app-header">
        <a className="brand" href="#top" onClick={() => setTool("gear")}>PC <span>Multitool</span></a>
        <nav aria-label="Tools">
          {tools.map(([key, name, title]) => (
            <button key={key} title={title} className={tool === key ? "active" : ""} onClick={() => setTool(key)}>{name}</button>
          ))}
        </nav>
      </header>
      {tool === "gear"     && <GearTool settings={settings} />}
      {tool === "crank"    && <CrankTool settings={settings} />}
      {tool === "wheel"    && <WheelTool settings={settings} />}
      {tool === "dyno"     && <DynoTool settings={settings} />}
      {tool === "piston"   && <PistonTool settings={settings} />}
      {tool === "settings" && <SettingsTool settings={settings} setSettings={setSettings} />}
      <CopyToastContainer />
    </div>
  );
}
