import { watch } from 'vue'

// Bump a version when legend metadata changes so WeakMap-cached tooltip HTML is dropped.
export function createVersionedTooltipCache(watchSource) {
  const wm = new WeakMap()
  let version = 0
  watch(watchSource, () => {
    version += 1
  })

  return {
    remember(key, render) {
      if (key && typeof key === 'object') {
        const cached = wm.get(key)
        if (cached?.version === version) return cached.html
        const html = render()
        wm.set(key, { version, html })
        return html
      }
      return render()
    }
  }
}
