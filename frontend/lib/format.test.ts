import { describe, it, expect } from "vitest";
import { fmt, tryParse, shortAddr } from "./format";

describe("fmt", () => {
  it("formats whole tokens", () => {
    expect(fmt(1_000_000_000_000_000_000n, 18, 2)).toBe("1.00");
  });
  it("truncates beyond fraction digits", () => {
    expect(fmt(1_234_567_890_000_000_000n, 18, 4)).toBe("1.2345");
  });
  it("handles undefined", () => {
    expect(fmt(undefined, 18)).toBe("—");
  });
});

describe("tryParse", () => {
  it("parses decimals", () => {
    expect(tryParse("1.5", 18)).toBe(1_500_000_000_000_000_000n);
  });
  it("rejects invalid input", () => {
    expect(tryParse("abc", 18)).toBeNull();
  });
});

describe("shortAddr", () => {
  it("shortens long address", () => {
    expect(shortAddr("0x1234567890abcdef1234567890abcdef12345678")).toBe("0x1234…5678");
  });
});
