import { createApp } from 'vue'

// The series name is the datum key. The legend only adds color once it has caught
// up; a missing legend entry must not blank the label.
export function seriesFromDatum(datum, indexKey, items = [], colors = []) {
  if (!datum || typeof datum !== 'object') return []
  return Object.entries(datum)
    .filter(([key]) => key !== indexKey)
    .map(([key, value], index) => {
      const legendReference = items.find((item) => item.name === key)
      return {
        ...legendReference,
        name: legendReference?.name || key,
        color: legendReference?.color ?? colors[index],
        value
      }
    })
}

export function renderTooltip(component, title, rows) {
  const host = document.createElement('div')
  createApp(component, { title, data: rows }).mount(host)
  return host.innerHTML
}
