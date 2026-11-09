// @vitest-environment jsdom
import { createApp, h, nextTick, reactive } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'

vi.mock('@unovis/vue', () => ({
  VisAxis: { name: 'VisAxis', render: () => null },
  VisGroupedBar: { name: 'VisGroupedBar', render: () => null },
  VisStackedBar: { name: 'VisStackedBar', render: () => null },
  VisXYContainer: { name: 'VisXYContainer', render: () => null },
  VisCrosshair: { name: 'VisCrosshair', render: () => null },
  VisTooltip: { name: 'VisTooltip', render: () => null }
}))

import BarChart from './BarChart.vue'

let app
let host

function mount(state) {
  host = document.createElement('div')
  document.body.append(host)
  app = createApp({
    setup() {
      return () =>
        h(BarChart, {
          data: state.data,
          categories: state.categories,
          index: 'week',
          type: 'stacked'
        })
    }
  })
  app.mount(host)
}

afterEach(() => {
  app?.unmount()
  host?.remove()
})

describe('BarChart legend', () => {
  it('shows tag names that arrive after mount', async () => {
    const state = reactive({
      data: [],
      categories: []
    })
    mount(state)
    expect(host.textContent).not.toContain('billing')

    state.categories = ['billing', 'bug']
    state.data = [{ week: '2026-08-03', billing: 9, bug: 7 }]
    await nextTick()

    expect(host.textContent).toContain('billing')
    expect(host.textContent).toContain('bug')
  })
})
