export const round3 = (value: number) => Math.round(value * 1000) / 1000;

export type GearResult = { full: number; half: number; offset: number; compressors: number; compressorValue: number };
export function calculateGear(teeth: number, compressors?: number): GearResult | null {
  // 1 and 2 are numerical singularities in the source formula.
  if (!Number.isInteger(teeth) || teeth < 3) return null;
  const full = 360 / teeth;
  const half = full / 2;
  const offset = Math.cos(half * Math.PI / 180) / Math.sin(full * Math.PI / 180) - 1;
  if (!Number.isFinite(offset)) return null;
  const automatic = Math.max(1, Math.ceil(offset));
  const count = compressors && Number.isInteger(compressors) && compressors > 0 ? compressors : automatic;
  return { full, half, offset, compressors: count, compressorValue: offset - (count - 1) };
}

export type CrankLayout = "Inline" | "V" | "Boxer";
export function calculateCrank(layout: CrankLayout, cylinders: number) {
  if (!Number.isInteger(cylinders) || cylinders < 1) return null;
  const positions = layout === "Inline" ? cylinders : Math.ceil(cylinders / 2);
  const angle = 360 / positions;
  return { positions, angle, phases: Array.from({ length: positions }, (_, i) => i * angle) };
}

export function calculateWheel(size: number) {
  if (!Number.isInteger(size) || size < 2) return null;
  const angleLock = 360 / size;
  const offset = 1 / Math.tan(Math.PI / size);
  const compressors = Math.max(1, Math.floor(offset) - 1);
  return { angleLock, centerAngle: size % 2 === 0 ? angleLock / 2 : null, offset, compressors, compressorValue: round3(Math.max(0, offset - compressors - 1)) };
}

export type DynoPoint = { sps: number; torque: number; rpm: number; bhp: number };
export type DynoGraphMode = "bhp-rpm" | "torque-sps";
export type DynoPlotPoint = { x: number; y: number };

export function dynoPlotCoordinates(points: DynoPoint[], mode: DynoGraphMode): DynoPlotPoint[] {
  return points.map(p => mode === "torque-sps" ? { x: p.sps, y: p.torque } : { x: p.rpm, y: p.bhp });
}

// Shape-preserving PCHIP sampling used in both graph modes. Sampling density
// follows the projected pixel span so exports stay smooth at 1600x900 without
// wasting work when the on-screen graph is smaller. Duplicate X values fall
// back to measured points rather than dividing by zero.
export function calculateDynoCurve(
  points: DynoPoint[],
  mode: DynoGraphMode,
  pixelWidth = 900,
  pixelHeight = 500,
): DynoPlotPoint[] {
  const mapped = dynoPlotCoordinates(points, mode);
  if (mapped.length <= 2) return mapped;

  const n = mapped.length, h: number[] = [], delta: number[] = [];
  let minX = mapped[0].x, maxX = mapped[0].x;
  let minY = mapped[0].y, maxY = mapped[0].y;
  for (const point of mapped) {
    minX = Math.min(minX, point.x);
    maxX = Math.max(maxX, point.x);
    minY = Math.min(minY, point.y);
    maxY = Math.max(maxY, point.y);
  }
  minY = Math.min(0, minY);
  if (maxX <= minX) maxX = minX + 100;
  if (maxY <= minY) maxY = minY + 100;

  for (let i = 0; i < n - 1; i++) {
    const width = mapped[i + 1].x - mapped[i].x;
    if (width <= 0) return mapped;
    h.push(width);
    delta.push((mapped[i + 1].y - mapped[i].y) / width);
  }

  const slopes = Array<number>(n).fill(0);
  const endpoint = (h0: number, h1: number, d0: number, d1: number) => {
    let d = ((2 * h0 + h1) * d0 - h0 * d1) / (h0 + h1);
    if (d === 0 || d0 === 0 || d * d0 < 0) return 0;
    if (d0 * d1 < 0 && Math.abs(d) > Math.abs(3 * d0)) d = 3 * d0;
    return d;
  };
  slopes[0] = endpoint(h[0], h[1], delta[0], delta[1]);
  for (let i = 1; i < n - 1; i++) {
    const d0 = delta[i - 1], d1 = delta[i];
    if (d0 === 0 || d1 === 0 || d0 * d1 <= 0) { slopes[i] = 0; continue; }
    const w1 = 2 * h[i] + h[i - 1], w2 = h[i] + 2 * h[i - 1];
    slopes[i] = (w1 + w2) / (w1 / d0 + w2 / d1);
  }
  slopes[n - 1] = endpoint(h[n - 2], h[n - 3], delta[n - 2], delta[n - 3]);

  const curve: DynoPlotPoint[] = [mapped[0]];
  const safeWidth = Math.max(1, pixelWidth), safeHeight = Math.max(1, pixelHeight);
  for (let i = 0; i < n - 1; i++) {
    const p1 = mapped[i], p2 = mapped[i + 1], width = p2.x - p1.x;
    const xSpan = Math.abs(width / (maxX - minX) * safeWidth);
    const ySpan = Math.abs((p2.y - p1.y) / (maxY - minY) * safeHeight);
    const pixelSpan = Math.max(xSpan, ySpan);
    const samplesPerSegment = Math.max(24, Math.min(8192, Math.ceil(pixelSpan * 1.5)));
    for (let sample = 1; sample <= samplesPerSegment; sample++) {
      const t = sample / samplesPerSegment, t2 = t * t, t3 = t2 * t;
      const y = (2*t3 - 3*t2 + 1)*p1.y + (t3 - 2*t2 + t)*width*slopes[i] + (-2*t3 + 3*t2)*p2.y + (t3 - t2)*width*slopes[i + 1];
      curve.push({
        x: p1.x + width*t,
        y: Math.max(Math.min(p1.y, p2.y), Math.min(Math.max(p1.y, p2.y), y)),
      });
    }
  }
  return curve;
}

// Local wall-clock timestamp used for downloaded graphs: YY-MM-DD-HH-MM-SS.
// sequence=0 is the normal filename; positive values avoid same-second clashes.
export function formatDynoExportFilename(date: Date, sequence = 0): string {
  const two = (value: number) => String(value).padStart(2, "0");
  const stamp = [
    two(date.getFullYear() % 100), two(date.getMonth() + 1), two(date.getDate()),
    two(date.getHours()), two(date.getMinutes()), two(date.getSeconds()),
  ].join("-");
  const suffix = sequence > 0 ? `-${String(sequence).padStart(2, "0")}` : "";
  return `dyno-${stamp}${suffix}.png`;
}

export function calculateDyno(rows: Array<{ sps: string; torque: string }>): DynoPoint[] {
  const points = rows.flatMap((row) => {
    const sps = Number(row.sps), torque = Number(row.torque);
    if (row.sps.trim() === "" || row.torque.trim() === "" || !Number.isFinite(sps) || !Number.isFinite(torque) || sps < 0) return [];
    const rpm = sps * 3.84;
    const bhp = torque * rpm / 52.52;
    if (!Number.isFinite(rpm) || !Number.isFinite(bhp)) return [];
    return [{ sps, torque, rpm, bhp }];
  }).sort((a, b) => a.rpm - b.rpm);
  return [{ sps: 0, torque: 0, rpm: 0, bhp: 0 }, ...points];
}

export function calculatePiston(distance: number, pistons: number, value: number) {
  if (!Number.isFinite(distance) || distance < 0 || !Number.isInteger(pistons) || pistons < 1 || !Number.isFinite(value) || value < 0) return null;
  return value * distance / pistons;
}
