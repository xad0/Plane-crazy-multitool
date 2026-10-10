import { useEffect, useMemo, useRef, useState } from "react";
import { calculateCrank, calculateDyno, calculateGear, calculatePiston, calculateWheel, DynoPoint, round3, CrankLayout } from "./lib/math";

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

function Card({ title, children, className = "" }: { title: string; children: React.ReactNode; className?: string }) { return <section className={`card ${className}`}><h2>{title}</h2>{children}</section>; }
function Field({ label, value, onChange, min, step = "1", hint }: { label: string; value: string; onChange: (value: string) => void; min?: number; step?: string; hint?: string }) {
  return <label className="field"><span>{label}</span><input type="number" inputMode="decimal" value={value} min={min} step={step} onChange={(e) => onChange(e.target.value)} />{hint && <small>{hint}</small>}</label>;
}
function Result({ label, value }: { label: string; value: string }) { return <div className="result"><span>{label}</span><strong>{value}</strong></div>; }

function GearTool({ settings }: { settings: Settings }) {
  const [teeth, setTeeth] = useState("6"), [compressors, setCompressors] = useState("1"), [manual, setManual] = useState(false);
  const toothValue = Number(teeth), result = calculateGear(toothValue, manual ? Number(compressors) : undefined);
  useEffect(() => { if (result && !manual) setCompressors(String(result.compressors)); }, [teeth, manual, result?.compressors]);
  const reference = useMemo(() => Array.from({ length: Math.min(200, Math.max(0, settings.referenceMax - settings.referenceMin + 1)) }, (_, i) => calculateGear(settings.referenceMin + i)!), [settings.referenceMin, settings.referenceMax]);
  return <ToolLayout title="Gear calculator" intro="">
    <Card title="Inputs"><Field label="Number of teeth" value={teeth} onChange={(v) => { setTeeth(v); setManual(false); }} min={3} /><div className="switch"><input aria-label="Automatic compressors" type="checkbox" checked={!manual} onChange={(e) => setManual(!e.target.checked)} /><span>Automatic compressors</span></div><Field label="Compressors" value={compressors} onChange={(v) => { setCompressors(v); setManual(true); }} min={1} /></Card>
    <Card title="Result">{!result ? <Notice tone="error">Enter a whole tooth count of 3 or more. Teeth 1–2 are mathematically invalid.</Notice> : <><div className="results-grid"><Result label="Full angle" value={fmt(result.full, "°")} /><Result label="Half angle" value={fmt(result.half, "°")} /><Result label="Offset" value={fmt(result.offset)} /><Result label="Compressor value" value={fmt(round3(result.compressorValue))} /></div>{result.compressorValue > 1 && <Notice tone="warn">MORE COMPRESSORS NEEDED</Notice>}</>}</Card>
    <Card title={`Gear reference · ${settings.referenceMin}–${settings.referenceMax} teeth`} className="wide"><div className="table-wrap"><table><thead><tr>{settings.columns.teeth && <th>Teeth</th>}{settings.columns.full && <th>Full angle</th>}{settings.columns.half && <th>Half angle</th>}{settings.columns.offset && <th>Offset</th>}{settings.columns.value && <th>Comp. value</th>}{settings.columns.compressors && <th>Compressors</th>}</tr></thead><tbody>{reference.map((row, i) => <tr key={settings.referenceMin + i}>{settings.columns.teeth && <td>{settings.referenceMin + i}</td>}{settings.columns.full && <td>{fmt(row.full, "°")}</td>}{settings.columns.half && <td>{fmt(row.half, "°")}</td>}{settings.columns.offset && <td>{fmt(row.offset)}</td>}{settings.columns.value && <td>{fmt(round3(row.compressorValue))}</td>}{settings.columns.compressors && <td>{row.compressors}</td>}</tr>)}</tbody></table></div></Card>
  </ToolLayout>;
}

function CrankTool() { const [layout, setLayout] = useState<CrankLayout>("Inline"), [cylinders, setCylinders] = useState("6"); const result = calculateCrank(layout, Number(cylinders)); return <ToolLayout title="Crank angle" intro="Find phase spacing for inline, V, and boxer engines."><Card title="Inputs"><label className="field"><span>Layout</span><select value={layout} onChange={(e) => setLayout(e.target.value as CrankLayout)}>{(["Inline", "V", "Boxer"] as const).map(x => <option key={x}>{x}</option>)}</select></label><Field label="Cylinders" value={cylinders} onChange={setCylinders} min={1} /></Card><Card title="Phase positions">{!result ? <Notice tone="error">Cylinder count must be positive.</Notice> : <><div className="results-grid"><Result label="Effective positions" value={String(result.positions)} /><Result label="Base phase spacing" value={fmt(result.angle, "°")} /></div><ol className="phase-list">{result.phases.map((phase, i) => <li key={i}><span>Position {i + 1}</span><strong>{fmt(phase, "°")}</strong></li>)}</ol></>}</Card></ToolLayout>; }
function WheelTool() { const [size, setSize] = useState("10"); const result = calculateWheel(Number(size)); return <ToolLayout title="Wheel calculator" intro="Wheel lock angles and compressor values use their own calculation rules."><Card title="Input"><Field label="Tire size (sm)" value={size} onChange={setSize} min={1} /></Card><Card title="Tire result">{!result ? <Notice tone="error">Enter a positive whole tire size.</Notice> : <div className="results-grid"><Result label="Angle lock" value={fmt(result.angleLock, "°")} /><Result label="Center angle" value={result.centerAngle === null ? "N/A" : fmt(result.centerAngle, "°")} /><Result label="Offset" value={fmt(round3(result.offset))} /><Result label="Compressors" value={String(result.compressors)} /><Result label="Compressor value" value={fmt(result.compressorValue)} /></div>}</Card></ToolLayout>; }
function PistonTool() { const [distance, setDistance] = useState("1"), [pistons, setPistons] = useState("2"), [value, setValue] = useState("3.106"); const length = calculatePiston(Number(distance), Number(pistons), Number(value)); return <ToolLayout title="Piston length" intro="Calculate piston length from crank distance, piston count, and value."><Card title="Inputs"><Field label="Crank distance" value={distance} onChange={setDistance} min={0} step="any" /><Field label="Amount of pistons" value={pistons} onChange={setPistons} min={1} /><Field label="Value" value={value} onChange={setValue} min={0} step="any" /></Card><Card title="Piston length">{length === null ? <Notice tone="error">Enter valid non-negative values and at least one piston.</Notice> : <><Result label="Calculated length" value={fmt(length)} />{length > 4 && <Notice tone="warn">VALUE EXCEEDS LIMIT</Notice>}</>}</Card></ToolLayout>; }

type DynoRow = { sps: string; torque: string };
function DynoTool() {
  const [rows, setRows] = useState<DynoRow[]>(Array.from({ length: 5 }, () => ({ sps: "", torque: "" })));
  const points = calculateDyno(rows);
  const measurements = points.slice(1);
  const update = (index: number, key: keyof DynoRow, value: string) => setRows(rows.map((r, i) => i === index ? { ...r, [key]: value } : r));
  const peaks = measurements.length ? { torque: measurements.reduce((a, b) => b.torque > a.torque ? b : a), sps: measurements.reduce((a, b) => b.sps > a.sps ? b : a), bhp: measurements.reduce((a, b) => b.bhp > a.bhp ? b : a) } : null;
  return <ToolLayout title="Dyno" intro="Enter SPS and torque samples to plot smooth BHP against RPM.">
    <Card title="Dyno data" className="dyno-data"><div className="dyno-table"><div className="dyno-head"><span>#</span><span>SPS</span><span>Torque</span></div>{rows.map((row, i) => <div className="dyno-row" key={i}><span>{i + 1}</span><input aria-label={`SPS row ${i + 1}`} inputMode="decimal" value={row.sps} onChange={(e) => update(i, "sps", e.target.value)} /><input aria-label={`Torque row ${i + 1}`} inputMode="decimal" value={row.torque} onChange={(e) => update(i, "torque", e.target.value)} /></div>)}</div><button className="secondary" onClick={() => setRows([...rows, { sps: "", torque: "" }])}>Add row</button></Card>
    <Card title="BHP vs RPM" className="dyno-chart"><DynoChart points={points} />{peaks ? <div className="peaks"><Result label="Peak torque" value={`${fmt(peaks.torque.torque)} @ ${fmt(peaks.torque.sps)} SPS`} /><Result label="Peak SPS" value={`${fmt(peaks.sps.sps)} @ ${fmt(peaks.sps.torque)} torque`} /><Result label="Peak BHP" value={`${fmt(peaks.bhp.bhp)} @ ${fmt(peaks.bhp.sps)} SPS`} /></div> : <p className="muted">Enter complete SPS and torque rows to begin.</p>}</Card>
  </ToolLayout>;
}

function DynoChart({ points }: { points: DynoPoint[] }) {
  const canvas = useRef<HTMLCanvasElement>(null), chartWrap = useRef<HTMLDivElement>(null), [hover, setHover] = useState<DynoPoint | null>(null), [fullscreen, setFullscreen] = useState(false);
  useEffect(() => {
    const el = canvas.current; if (!el) return;
    const draw = () => drawDyno(el, points);
    const observer = new ResizeObserver(draw); observer.observe(el); draw(); return () => observer.disconnect();
  }, [points]);
  useEffect(() => {
    const sync = () => setFullscreen(document.fullscreenElement === chartWrap.current);
    document.addEventListener("fullscreenchange", sync);
    sync();
    return () => document.removeEventListener("fullscreenchange", sync);
  }, []);
  const pointer = (event: React.PointerEvent<HTMLCanvasElement>) => {
    const rect = event.currentTarget.getBoundingClientRect(); if (!points.length) return;
    const ratio = Math.max(0, Math.min(1, (event.clientX - rect.left - 48) / Math.max(1, rect.width - 72)));
    const min = points[0].rpm, max = points.at(-1)!.rpm;
    setHover(points.reduce((nearest, point) => Math.abs(point.rpm - (min + ratio * (max - min))) < Math.abs(nearest.rpm - (min + ratio * (max - min))) ? point : nearest));
  };
  const download = () => { const el = canvas.current; if (!el || !points.length) return; const a = document.createElement("a"); a.download = "pc-multitool-dyno.png"; a.href = el.toDataURL("image/png"); a.click(); };
  const toggleFullscreen = async () => { const wrap = chartWrap.current; if (!wrap) return; try { if (document.fullscreenElement === wrap) await document.exitFullscreen(); else await wrap.requestFullscreen(); } catch { setFullscreen(false); } };
  return <div ref={chartWrap} className="chart-wrap"><canvas ref={canvas} className="chart" role="img" aria-label="Smooth BHP versus RPM Dyno graph" onPointerMove={pointer} onPointerLeave={() => setHover(null)} />{hover && <output className="chart-tip">{fmt(hover.rpm)} RPM · {fmt(hover.bhp)} BHP</output>}<div className="chart-actions"><button className="secondary" onClick={toggleFullscreen}>{fullscreen ? "Exit fullscreen" : "Fullscreen"}</button><button className="secondary" disabled={!points.length} onClick={download}>Save PNG</button></div></div>;
}

function drawDyno(canvas: HTMLCanvasElement, points: DynoPoint[]) {
  const rect = canvas.getBoundingClientRect(), dpr = Math.min(window.devicePixelRatio || 1, 2), w = Math.max(280, rect.width), h = Math.max(260, rect.height);
  canvas.width = w * dpr; canvas.height = h * dpr; const ctx = canvas.getContext("2d")!; ctx.scale(dpr, dpr); ctx.clearRect(0, 0, w, h);
  const css = getComputedStyle(document.documentElement), muted = css.getPropertyValue("--muted"), border = css.getPropertyValue("--border"), accent = css.getPropertyValue("--accent"), text = css.getPropertyValue("--text");
  const left = 52, right = 18, top = 16, bottom = 36, pw = w - left - right, ph = h - top - bottom;
  ctx.font = "12px system-ui"; ctx.strokeStyle = border; ctx.fillStyle = muted; ctx.lineWidth = 1;
  if (!points.length) { ctx.textAlign = "center"; ctx.fillText("Enter SPS and torque data", w / 2, h / 2); return; }
  let minX = points[0].rpm, maxX = points.at(-1)!.rpm, minY = Math.min(0, ...points.map(p => p.bhp)), maxY = Math.max(...points.map(p => p.bhp)); if (maxX === minX) maxX += 100; if (maxY === minY) maxY += 100;
  const x = (v: number) => left + (v - minX) / (maxX - minX) * pw, y = (v: number) => top + (1 - (v - minY) / (maxY - minY)) * ph;
  for (let i = 0; i <= 5; i++) { const gy = top + ph * i / 5, gx = left + pw * i / 5; ctx.beginPath(); ctx.moveTo(left, gy); ctx.lineTo(w - right, gy); ctx.moveTo(gx, top); ctx.lineTo(gx, h - bottom); ctx.stroke(); ctx.textAlign = "right"; ctx.fillText((maxY - (maxY - minY) * i / 5).toFixed(0), left - 7, gy + 4); ctx.textAlign = "center"; ctx.fillText((minX + (maxX - minX) * i / 5).toFixed(0), gx, h - 14); }
  ctx.strokeStyle = text; ctx.beginPath(); ctx.moveTo(left, top); ctx.lineTo(left, h - bottom); ctx.lineTo(w - right, h - bottom); ctx.stroke();
  ctx.strokeStyle = accent; ctx.lineWidth = 3; ctx.lineJoin = "round"; ctx.lineCap = "round"; ctx.beginPath();
  if (points.length === 1) { ctx.arc(x(points[0].rpm), y(points[0].bhp), 4, 0, Math.PI * 2); ctx.stroke(); return; }
  ctx.moveTo(x(points[0].rpm), y(points[0].bhp));
  // Catmull–Rom converted to cubic Bézier curves: smooth and passes through each measurement.
  for (let i = 0; i < points.length - 1; i++) { const p0 = points[Math.max(0, i - 1)], p1 = points[i], p2 = points[i + 1], p3 = points[Math.min(points.length - 1, i + 2)]; ctx.bezierCurveTo(x(p1.rpm + (p2.rpm - p0.rpm) / 6), y(p1.bhp + (p2.bhp - p0.bhp) / 6), x(p2.rpm - (p3.rpm - p1.rpm) / 6), y(p2.bhp - (p3.bhp - p1.bhp) / 6), x(p2.rpm), y(p2.bhp)); }
  ctx.stroke(); ctx.fillStyle = accent; points.forEach(p => { ctx.beginPath(); ctx.arc(x(p.rpm), y(p.bhp), 3.5, 0, Math.PI * 2); ctx.fill(); });
}

function SettingsTool({ settings, setSettings }: { settings: Settings; setSettings: React.Dispatch<React.SetStateAction<Settings>> }) {
  const [section, setSection] = useState<"appearance" | "layout" | "reference" | "calculator">("appearance");
  const columns: Array<[string, string]> = [["teeth", "Teeth"], ["full", "Full angle"], ["half", "Half angle"], ["offset", "Offset"], ["value", "Compressor value"], ["compressors", "Compressors"]];
  const calculator: Array<[string, string]> = [["teeth", "Number of teeth"], ["compressors", "Compressors"], ["full", "Full angle"], ["half", "Half angle"], ["offset", "Offset"], ["value", "Compressor value"], ["warnings", "Warnings"]];
  const toggle = (group: "columns" | "calculator", key: string) => setSettings(s => ({ ...s, [group]: { ...s[group], [key]: !s[group][key] } }));
  return <ToolLayout title="Settings" intro="The same organization as the desktop app, adapted for browser controls.">
    <div className="settings-tabs" role="tablist">{([ ["appearance", "Appearance"], ["layout", "Layout"], ["reference", "Reference"], ["calculator", "Calculator"] ] as const).map(([key, label]) => <button key={key} role="tab" aria-selected={section === key} className={section === key ? "active" : ""} onClick={() => setSection(key)}>{label}</button>)}</div>
    {section === "appearance" && <Card title="Theme"><label className="field"><span>Theme family</span><select value={settings.theme.startsWith("catppuccin") ? "catppuccin" : settings.theme} onChange={(e) => { const theme = e.target.value === "catppuccin" ? "catppuccin-mocha" : e.target.value as Theme; setSettings(s => ({ ...s, theme })); }}><option value="catppuccin">Catppuccin</option><option value="nord">Nord</option><option value="gruvbox">Gruvbox</option><option value="solid">Solid</option></select></label>{settings.theme.startsWith("catppuccin") && <label className="field"><span>Catppuccin flavor</span><select value={settings.theme} onChange={(e) => setSettings(s => ({ ...s, theme: e.target.value as Theme }))}><option value="catppuccin-latte">Latte</option><option value="catppuccin-frappe">Frappé</option><option value="catppuccin-macchiato">Macchiato</option><option value="catppuccin-mocha">Mocha</option></select></label>}<label className="field"><span>Accent</span><select value={settings.accent} onChange={(e) => setSettings(s => ({ ...s, accent: e.target.value }))}>{["green", "blue", "purple", "red", "orange", "cyan", "pink"].map(a => <option key={a}>{a}</option>)}</select></label></Card>}
    {section === "layout" && <><Card title="Gear calculator layout"><label className="field"><span>Desktop arrangement</span><select value={settings.gearLayout} onChange={(e) => setSettings(s => ({ ...s, gearLayout: e.target.value as Settings["gearLayout"] }))}>{[["automatic", "Automatic"], ["calculator-left", "Calculator left"], ["calculator-right", "Calculator right"], ["calculator-only", "Calculator only"], ["reference-only", "Reference only"], ["stacked", "Stacked"]].map(([v, l]) => <option value={v} key={v}>{l}</option>)}</select></label></Card><Card title="Other tools"><SideSelect label="Crank results" value={settings.crankSide} change={(v) => setSettings(s => ({ ...s, crankSide: v }))} /><SideSelect label="Wheel results" value={settings.wheelSide} change={(v) => setSettings(s => ({ ...s, wheelSide: v }))} /><SideSelect label="Dyno graph" value={settings.dynoSide} change={(v) => setSettings(s => ({ ...s, dynoSide: v }))} /><SideSelect label="Piston results" value={settings.pistonSide} change={(v) => setSettings(s => ({ ...s, pistonSide: v }))} /></Card></>}
    {section === "reference" && <><Card title="Reference chart"><label className="switch"><input type="checkbox" checked={settings.referenceEnabled} onChange={() => setSettings(s => ({ ...s, referenceEnabled: !s.referenceEnabled }))} /><span>Show reference chart</span></label><Field label="Smallest gear" value={String(settings.referenceMin)} onChange={(v) => setSettings(s => ({ ...s, referenceMin: Math.max(3, Number(v) || 3) }))} min={3} /><Field label="Maximum gear" value={String(settings.referenceMax)} onChange={(v) => setSettings(s => ({ ...s, referenceMax: Math.max(s.referenceMin, Math.min(202, Number(v) || s.referenceMin)) }))} min={settings.referenceMin} /></Card><Card title="Reference columns"><div className="checks">{columns.map(([key, label]) => <label key={key}><input type="checkbox" checked={settings.columns[key]} onChange={() => toggle("columns", key)} /> {label}</label>)}</div></Card></>}
    {section === "calculator" && <Card title="Gear calculator fields"><div className="checks">{calculator.map(([key, label]) => <label key={key}><input type="checkbox" checked={settings.calculator[key]} onChange={() => toggle("calculator", key)} /> {label}</label>)}</div></Card>}
  </ToolLayout>;
}
function SideSelect({ label, value, change }: { label: string; value: Side; change: (value: Side) => void }) { return <label className="field"><span>{label}</span><select value={value} onChange={(e) => change(e.target.value as Side)}><option value="right">Right</option><option value="left">Left</option></select></label>; }
function ToolLayout({ title, children }: { title: string; intro?: string; children: React.ReactNode }) { return <main><header className="page-header"><h1>{title}</h1></header><div className="tool-grid">{children}</div></main>; }
function Notice({ tone, children }: { tone: "warn" | "error"; children: React.ReactNode }) { return <p className={`notice ${tone}`}>{children}</p>; }
export function App() { const [settings, setSettings] = useStoredSettings(), [tool, setTool] = useState<Tool>("gear"); const theme = settings.theme in palette ? settings.theme : "catppuccin-mocha"; useEffect(() => { const root = document.documentElement; root.dataset.theme = theme; root.dataset.accent = settings.accent; Object.entries(palette[theme]).forEach(([name, value]) => root.style.setProperty(name, value)); document.querySelector('meta[name="theme-color"]')?.setAttribute("content", palette[theme]["--bg"]); }, [theme, settings.accent]); return <div className="app-shell"><header className="app-header"><a className="brand" href="#top" onClick={() => setTool("gear")}>PC <span>Multitool</span></a><nav aria-label="Tools">{tools.map(([key, name, title]) => <button key={key} title={title} className={tool === key ? "active" : ""} onClick={() => setTool(key)}>{name}</button>)}</nav></header>{tool === "gear" && <GearTool settings={settings} />}{tool === "crank" && <CrankTool />}{tool === "wheel" && <WheelTool />}{tool === "dyno" && <DynoTool />}{tool === "piston" && <PistonTool />}{tool === "settings" && <SettingsTool settings={settings} setSettings={setSettings} />}</div>; }
