// @vitest-environment jsdom
import { nextTick, reactive } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mountChartWithLegend } from '../chart/chartLegendTestMount.js'

vi.mock('@unovis/vue', () => ({
  VisAxis: { name: 'VisAxis', render: () => null },
  VisLine: { name: 'VisLine', render: () => null },
  VisXYContainer: { name: 'VisXYContainer', render: () => null },
  VisCrosshair: { name: 'VisCrosshair', render: () => null },
  VisTooltip: { name: 'VisTooltip', render: () => null }
}))

import LineChart from './LineChart.vue'

let mounted

afterEach(() => {
  mounted?.unmount()
})

describe('LineChart legend', () => {
  it('shows status names that arrive after mount', async () => {
    const state = reactive({
      data: [],
      categories: []
    })
    mounted = mountChartWithLegend(LineChart, state, (s) => ({
      data: s.data,
      categories: s.categories,
      index: 'date'
    }))
    const { host } = mounted
    expect(host.textContent).not.toContain('Open')

    state.categories = ['Open', 'Waiting on customer']
    state.data = [{ date: '2026-07-13', Open: 120, 'Waiting on customer': 40 }]
    await nextTick()

    expect(host.textContent).toContain('Open')
    expect(host.textContent).toContain('Waiting on customer')
  })
})
