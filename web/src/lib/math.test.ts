import { describe, expect, it } from "vitest";
import { calculateCrank, calculateDyno, calculateDynoCurve, dynoPlotCoordinates, formatDynoExportFilename, calculateGear, calculatePiston, calculateWheel, round3 } from "./math";

describe("Go-reference math", () => {
  it("matches gear golden values", () => {
    expect(calculateGear(4)?.offset).toBeCloseTo(-0.2928932188, 9);
    expect(calculateGear(16)?.offset).toBeCloseTo(1.5629154477, 9);
    expect(calculateGear(100)?.compressors).toBe(15);
    expect(calculateGear(1)).toBeNull();
  });
  it("rounds like Go", () => expect(round3(-0.2928932188)).toBe(-0.293));
  it("keeps wheel compressor behavior separate", () => {
    expect(calculateWheel(10)).toMatchObject({ compressors: 2, centerAngle: 18 });
    expect(calculateWheel(1)).toBeNull();
    expect(calculateWheel(2)).not.toBeNull();
  });
  it("validates crank layouts and piston inputs", () => {
    expect(calculateCrank("V", 5)?.phases).toEqual([0, 120, 240]);
    expect(calculateCrank("Boxer", 5)?.phases).toEqual([0, 120, 240]);
    expect(calculateCrank("Boxer", 6)?.phases).toEqual([0, 120, 240]);
    expect(calculatePiston(2, 2, 3.106)).toBe(3.106);
  });
  it("skips incomplete and negative-SPS dyno rows", () => {
    const points = calculateDyno([{ sps: "10", torque: "20" }, { sps: "-1", torque: "4" }, { sps: "", torque: "2" }]);
    expect(points).toHaveLength(2);
    expect(points[0]).toEqual({ sps: 0, torque: 0, rpm: 0, bhp: 0 });
  });
  it("skips dyno rows whose derived values overflow", () => {
    const points = calculateDyno([{ sps: "1.7976931348623157e+308", torque: "1" }, { sps: "10", torque: "1.7976931348623157e+308" }]);
    expect(points).toEqual([{ sps: 0, torque: 0, rpm: 0, bhp: 0 }]);
  });
  it("maps both dyno graph modes to their correct axes", () => {
    const points = calculateDyno([{ sps: "10", torque: "4" }, { sps: "20", torque: "8" }]);
    expect(dynoPlotCoordinates(points, "bhp-rpm")[1]).toMatchObject({ x: 38.4, y: points[1].bhp });
    expect(dynoPlotCoordinates(points, "torque-sps")[1]).toEqual({ x: 10, y: 4 });
  });
  it("samples a bounded smooth dyno curve in torque/SPS mode", () => {
    const points = calculateDyno([{ sps: "10", torque: "0" }, { sps: "20", torque: "12" }, { sps: "30", torque: "4" }, { sps: "40", torque: "20" }]);
    const curve = calculateDynoCurve(points, "torque-sps");
    expect(curve.length).toBeGreaterThan(points.length);
    for (let i = 0; i < curve.length; i++) {
      expect(Number.isFinite(curve[i].x) && Number.isFinite(curve[i].y)).toBe(true);
    }
    const xs = points.map(p => p.sps), ys = points.map(p => p.torque);
    for (const sample of curve) {
      const segment = Math.min(xs.length - 2, Math.max(0, xs.findIndex((x, i) => i < xs.length - 1 && sample.x >= x && sample.x <= xs[i + 1])));
      expect(sample.y).toBeGreaterThanOrEqual(Math.min(ys[segment], ys[segment + 1]) - 1e-9);
      expect(sample.y).toBeLessThanOrEqual(Math.max(ys[segment], ys[segment + 1]) + 1e-9);
    }
  });

  it("adapts curve sampling to the rendered pixel size", () => {
    const points = calculateDyno([
      { sps: "10", torque: "20" }, { sps: "20", torque: "8" },
      { sps: "30", torque: "18" }, { sps: "40", torque: "12" },
    ]);
    const small = calculateDynoCurve(points, "bhp-rpm", 160, 80);
    const large = calculateDynoCurve(points, "bhp-rpm", 1600, 800);
    expect(large.length).toBeGreaterThan(small.length);
    expect(large.every(point => Number.isFinite(point.x) && Number.isFinite(point.y))).toBe(true);
  });

  it("formats compact local-time export names and same-second suffixes", () => {
    const date = new Date(2026, 9, 22, 22, 55, 10);
    expect(formatDynoExportFilename(date)).toBe("dyno-26-10-22-22-55-10.png");
    expect(formatDynoExportFilename(date, 1)).toBe("dyno-26-10-22-22-55-10-01.png");
  });
});
