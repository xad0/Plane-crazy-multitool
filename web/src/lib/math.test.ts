import { describe, expect, it } from "vitest";
import { calculateCrank, calculateDyno, calculateGear, calculatePiston, calculateWheel, round3 } from "./math";

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
    expect(calculateCrank("V", 5)).toBeNull();
    expect(calculateCrank("Boxer", 6)?.phases).toEqual([0, 120, 240]);
    expect(calculatePiston(2, 2, 3.106)).toBe(3.106);
  });
  it("skips incomplete and negative-SPS dyno rows", () => {
    expect(calculateDyno([{ sps: "10", torque: "20" }, { sps: "-1", torque: "4" }, { sps: "", torque: "2" }])).toHaveLength(1);
  });
});
