import { describe, expect, it } from 'vitest'
import { seriesFromDatum } from './tooltipSeries'

describe('seriesFromDatum', () => {
  it('keeps the status name when the legend has not caught up', () => {
    const rows = seriesFromDatum(
      { date: '2026-07-13', Open: 120, 'Waiting on customer': 40 },
      'date',
      [],
      ['green', 'mint']
    )
    expect(rows.map((row) => row.name)).toEqual(['Open', 'Waiting on customer'])
    expect(rows.map((row) => row.color)).toEqual(['green', 'mint'])
    expect(rows.map((row) => row.value)).toEqual([120, 40])
  })

  it('uses the legend color when the series is already listed', () => {
    const rows = seriesFromDatum({ date: '2026-07-13', Open: 120 }, 'date', [
      { name: 'Open', color: 'hsl(green)', inactive: false }
    ])
    expect(rows).toEqual([
      { name: 'Open', color: 'hsl(green)', inactive: false, value: 120 }
    ])
  })
})
