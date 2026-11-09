// @vitest-environment jsdom
import { createApp, h, nextTick, reactive } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'

vi.mock('@unovis/vue', () => ({
  VisAxis: { name: 'VisAxis', render: () => null },
  VisLine: { name: 'VisLine', render: () => null },
  VisXYContainer: { name: 'VisXYContainer', render: () => null },
  VisCrosshair: { name: 'VisCrosshair', render: () => null },
  VisTooltip: { name: 'VisTooltip', render: () => null }
}))

import LineChart from './LineChart.vue'

let app
let host

function mount(state) {
  host = document.createElement('div')
  document.body.append(host)
  app = createApp({
    setup() {
      return () =>
        h(LineChart, {
          data: state.data,
          categories: state.categories,
          index: 'date'
        })
    }
  })
  app.mount(host)
}

afterEach(() => {
  app?.unmount()
  host?.remove()
})

describe('LineChart legend', () => {
  it('shows status names that arrive after mount', async () => {
    const state = reactive({
      data: [],
      categories: []
    })
    mount(state)
    expect(host.textContent).not.toContain('Open')

    state.categories = ['Open', 'Waiting on customer']
    state.data = [{ date: '2026-07-13', Open: 120, 'Waiting on customer': 40 }]
    await nextTick()

    expect(host.textContent).toContain('Open')
    expect(host.textContent).toContain('Waiting on customer')
  })
})
