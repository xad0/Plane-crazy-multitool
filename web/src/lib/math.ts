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
  if (!Number.isInteger(cylinders) || cylinders < 1 || (layout !== "Inline" && cylinders % 2 !== 0)) return null;
  const positions = layout === "Inline" ? cylinders : cylinders / 2;
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
export function calculateDyno(rows: Array<{ sps: string; torque: string }>): DynoPoint[] {
  return rows.flatMap((row) => {
    const sps = Number(row.sps), torque = Number(row.torque);
    if (row.sps.trim() === "" || row.torque.trim() === "" || !Number.isFinite(sps) || !Number.isFinite(torque) || sps < 0) return [];
    const rpm = sps * 3.84;
    return [{ sps, torque, rpm, bhp: torque * rpm / 52.52 }];
  }).sort((a, b) => a.rpm - b.rpm);
}

export function calculatePiston(distance: number, pistons: number, value: number) {
  if (!Number.isFinite(distance) || distance < 0 || !Number.isInteger(pistons) || pistons < 1 || !Number.isFinite(value) || value < 0) return null;
  return value * distance / pistons;
}
