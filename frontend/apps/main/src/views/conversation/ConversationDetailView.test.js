// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createApp, h, nextTick, reactive, ref } from 'vue'

const { store } = vi.hoisted(() => ({ store: { value: null } }))
vi.mock('@main/stores/conversation', () => ({ useConversationStore: () => store.value }))
vi.mock('@main/composables/useEmitter', () => ({ useEmitter: () => ({ on: vi.fn(), off: vi.fn() }) }))
vi.mock('@shared-ui/composables', () => ({ useIsMobile: () => ref(true) }))
vi.mock('@main/features/conversation/Conversation.vue', () => ({
  default: { render: () => h('div', { class: 'conversation' }) }
}))
vi.mock('@main/features/conversation/sidebar/ConversationSideBar.vue', () => ({ default: { render: () => null } }))
vi.mock('@shared-ui/components/ui/sheet', () => ({
  Sheet: { render: () => null },
  SheetContent: { render: () => null },
  SheetTitle: { render: () => null }
}))
vi.mock('@shared-ui/components/ui/resizable', () => ({
  ResizablePanelGroup: { render: () => null },
  ResizablePanel: { render: () => null },
  ResizableHandle: { render: () => null }
}))
import ConversationDetailView from './ConversationDetailView.vue'

let app
let root
afterEach(() => {
  app?.unmount()
  root?.remove()
})

const mount = async (props) => {
  store.value = reactive({
    current: { uuid: 'previous' },
    conversation: { loading: false },
    messages: { loading: false },
    fetchConversation: vi.fn(),
    fetchMessages: vi.fn()
  })
  root = document.createElement('div')
  document.body.appendChild(root)
  app = createApp(ConversationDetailView, props)
  app.mount(root)
  await nextTick()
}
const shown = () => !!root.querySelector('.conversation')

describe('ConversationDetailView', () => {
  it('keeps the previous conversation on screen while the next one loads by default', async () => {
    await mount({ uuid: 'next' })
    expect(shown()).toBe(true)
  })

  it('with hideStale, shows nothing until the requested conversation has loaded', async () => {
    await mount({ uuid: 'next', hideStale: true })
    expect(shown()).toBe(false)

    store.value.current = { uuid: 'next' }
    await nextTick()
    expect(shown()).toBe(true)
  })
})
