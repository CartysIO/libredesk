import { effectScope, nextTick, reactive } from 'vue'
import { describe, expect, it } from 'vitest'
import { syncLegendItems, useLegendItems } from './legendItems'

describe('syncLegendItems', () => {
  it('uses the category name and the matching color', () => {
    expect(syncLegendItems(['billing', 'bug'], ['green', 'mint'])).toEqual([
      { name: 'billing', color: 'green', inactive: false },
      { name: 'bug', color: 'mint', inactive: false }
    ])
  })

  it('keeps a hidden series hidden when that name is still present', () => {
    const previous = [
      { name: 'billing', color: 'green', inactive: true },
      { name: 'bug', color: 'mint', inactive: false }
    ]
    expect(syncLegendItems(['bug', 'billing', 'outage'], ['a', 'b', 'c'], previous)).toEqual([
      { name: 'bug', color: 'a', inactive: false },
      { name: 'billing', color: 'b', inactive: true },
      { name: 'outage', color: 'c', inactive: false }
    ])
  })
})

describe('useLegendItems', () => {
  function mount(state) {
    let items
    const scope = effectScope()
    scope.run(() => {
      items = useLegendItems(
        () => state.categories,
        () => state.categories.map((_, i) => `c${i}`)
      )
    })
    return { items, stop: () => scope.stop() }
  }

  it('fills names after categories arrive', async () => {
    const state = reactive({ categories: [] })
    const { items, stop } = mount(state)
    expect(items.value).toEqual([])

    state.categories = ['billing', 'bug']
    await nextTick()

    expect(items.value).toEqual([
      { name: 'billing', color: 'c0', inactive: false },
      { name: 'bug', color: 'c1', inactive: false }
    ])
    stop()
  })

  it('picks up a series added in place', async () => {
    const state = reactive({ categories: ['Open'] })
    const { items, stop } = mount(state)

    state.categories.push('Waiting on customer')
    await nextTick()

    expect(items.value.map((item) => item.name)).toEqual(['Open', 'Waiting on customer'])
    stop()
  })
})
