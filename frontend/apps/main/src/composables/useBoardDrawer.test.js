// @vitest-environment jsdom
import { describe, it, expect, beforeEach } from 'vitest'
import { useBoardDrawer } from './useBoardDrawer'

describe('useBoardDrawer', () => {
  beforeEach(() => {
    localStorage.clear()
    useBoardDrawer().resetExpanded()
  })

  it('toggles and resets expand state', () => {
    const d = useBoardDrawer()
    expect(d.expanded.value).toBe(false)
    d.toggleExpanded()
    expect(d.expanded.value).toBe(true)
    d.toggleExpanded()
    expect(d.expanded.value).toBe(false)
    d.toggleExpanded()
    d.resetExpanded()
    expect(d.expanded.value).toBe(false)
  })

  it('remembers the width', () => {
    useBoardDrawer().setWidth(812.4)
    expect(useBoardDrawer().width.value).toBe(812.4)
    expect(localStorage.getItem('conversationBoardDrawerWidth')).toBe('812')
  })
})
