import { describe, expect, it } from 'vitest'
import { donutArcPath, donutSegmentColors, polarToCartesian } from './donutSegments.js'

describe('donutArcPath', () => {
  it('returns a path for a full ring (single category)', () => {
    const d = donutArcPath(0, 360)
    expect(d.length).toBeGreaterThan(0)
    expect(d).toMatch(/^M /)
  })

  it('returns empty for zero sweep', () => {
    expect(donutArcPath(10, 10)).toBe('')
  })
})

describe('donutSegmentColors', () => {
  it('returns one distinct color per slice', () => {
    const colors = donutSegmentColors(3)
    expect(colors).toHaveLength(3)
    expect(new Set(colors).size).toBe(3)
  })
})

describe('polarToCartesian', () => {
  it('places 0° at the top of the chart', () => {
    const p = polarToCartesian(0, 0, 10, 0)
    expect(p.x).toBeCloseTo(0)
    expect(p.y).toBeCloseTo(-10)
  })
})
