import { ref, watch } from 'vue'

// Series names often arrive after the chart mounts. Rebuild the legend from the
// current categories and keep a series hidden if the user already toggled it.
export function syncLegendItems(categories, palette, previous = []) {
  const inactiveByName = new Map(previous.map((item) => [item.name, !!item.inactive]))
  return (categories || []).map((category, i) => ({
    name: category,
    color: palette?.[i],
    inactive: inactiveByName.get(category) ?? false
  }))
}

export function useLegendItems(categories, colors) {
  const legendItems = ref([])
  watch(
    [categories, colors],
    ([nextCategories, palette]) => {
      legendItems.value = syncLegendItems(nextCategories, palette, legendItems.value)
    },
    { immediate: true, deep: true }
  )
  return legendItems
}
