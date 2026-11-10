// @vitest-environment jsdom
import { nextTick, reactive } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mountChartWithLegend } from '../chart/chartLegendTestMount.js'

vi.mock('@unovis/vue', () => ({
  VisAxis: { name: 'VisAxis', render: () => null },
  VisGroupedBar: { name: 'VisGroupedBar', render: () => null },
  VisStackedBar: { name: 'VisStackedBar', render: () => null },
  VisXYContainer: { name: 'VisXYContainer', render: () => null },
  VisCrosshair: { name: 'VisCrosshair', render: () => null },
  VisTooltip: { name: 'VisTooltip', render: () => null }
}))

import BarChart from './BarChart.vue'

let mounted

afterEach(() => {
  mounted?.unmount()
})

describe('BarChart legend', () => {
  it('shows tag names that arrive after mount', async () => {
    const state = reactive({
      data: [],
      categories: []
    })
    mounted = mountChartWithLegend(BarChart, state, (s) => ({
      data: s.data,
      categories: s.categories,
      index: 'week',
      type: 'stacked'
    }))
    const { host } = mounted
    expect(host.textContent).not.toContain('billing')

    state.categories = ['billing', 'bug']
    state.data = [{ week: '2026-08-03', billing: 9, bug: 7 }]
    await nextTick()

    expect(host.textContent).toContain('billing')
    expect(host.textContent).toContain('bug')
  })
})
