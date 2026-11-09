import { createApp, h } from 'vue'

export function mountChartWithLegend(ChartComponent, state, chartProps) {
  const host = document.createElement('div')
  document.body.append(host)
  const app = createApp({
    setup() {
      return () => h(ChartComponent, chartProps(state))
    }
  })
  app.mount(host)
  return {
    host,
    unmount() {
      app.unmount()
      host.remove()
    }
  }
}
