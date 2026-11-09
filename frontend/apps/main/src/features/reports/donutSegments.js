/** Distinct hues so each slice stays readable on light and dark backgrounds. */
const SEGMENT_HUES = [152, 25, 4, 48, 210, 265, 320]

export function donutSegmentColors(count) {
  return Array.from({ length: count }, (_, i) => {
    const hue = SEGMENT_HUES[i % SEGMENT_HUES.length]
    return `hsl(${hue} 58% 44%)`
  })
}

export function polarToCartesian(cx, cy, r, angleDeg) {
  const rad = ((angleDeg - 90) * Math.PI) / 180
  return { x: cx + r * Math.cos(rad), y: cy + r * Math.sin(rad) }
}

function singleDonutArc(startAngle, endAngle, outerR, innerR) {
  const start = polarToCartesian(0, 0, outerR, endAngle)
  const end = polarToCartesian(0, 0, outerR, startAngle)
  const innerStart = polarToCartesian(0, 0, innerR, endAngle)
  const innerEnd = polarToCartesian(0, 0, innerR, startAngle)
  const sweep = endAngle - startAngle
  const largeArc = sweep <= 180 ? '0' : '1'
  return [
    `M ${start.x} ${start.y}`,
    `A ${outerR} ${outerR} 0 ${largeArc} 0 ${end.x} ${end.y}`,
    `L ${innerEnd.x} ${innerEnd.y}`,
    `A ${innerR} ${innerR} 0 ${largeArc} 1 ${innerStart.x} ${innerStart.y}`,
    'Z'
  ].join(' ')
}

/** SVG elliptical arcs cannot sweep a full 360° in one command; split when needed. */
export function donutArcPath(startAngle, endAngle, outerR = 70, innerR = 42) {
  const sweep = endAngle - startAngle
  if (sweep <= 0) return ''
  if (sweep >= 359.99) {
    const mid = startAngle + 180
    return `${singleDonutArc(startAngle, mid, outerR, innerR)} ${singleDonutArc(mid, endAngle, outerR, innerR)}`
  }
  return singleDonutArc(startAngle, endAngle, outerR, innerR)
}
