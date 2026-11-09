import { effectScope, nextTick, ref } from 'vue'
import { describe, expect, it } from 'vitest'
import { createVersionedTooltipCache } from './tooltipCache'

describe('createVersionedTooltipCache', () => {
  it('re-renders after legend items change', async () => {
    const items = ref([{ name: 'Open', color: 'green', inactive: false }])
    let calls = 0
    const key = {}
    let cache
    const scope = effectScope()
    scope.run(() => {
      cache = createVersionedTooltipCache(() => items.value)
    })

    expect(cache.remember(key, () => `v${++calls}`)).toBe('v1')
    expect(cache.remember(key, () => `v${++calls}`)).toBe('v1')

    items.value = [...items.value, { name: 'Closed', color: 'mint', inactive: false }]
    await nextTick()

    expect(cache.remember(key, () => `v${++calls}`)).toBe('v2')
    scope.stop()
  })
})
