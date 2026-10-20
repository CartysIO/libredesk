// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createApp, h, nextTick, reactive } from 'vue'

const { store, api, can } = vi.hoisted(() => ({
  store: { value: null },
  can: { value: true },
  api: {
    getLinearIssues: vi.fn(),
    linkLinearIssue: vi.fn(),
    unlinkLinearIssue: vi.fn()
  }
}))
vi.mock('@/stores/conversation', () => ({ useConversationStore: () => store.value }))
vi.mock('@/stores/user', () => ({ useUserStore: () => ({ can: () => can.value }) }))
vi.mock('@/composables/useEmitter', () => ({ useEmitter: () => ({ emit: vi.fn() }) }))
vi.mock('@/api', () => ({ default: api }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (k) => k }) }))
import { Accordion } from '@shared-ui/components/ui/accordion'
import ConversationSideBarLinear from './ConversationSideBarLinear.vue'

let app
let root
afterEach(() => {
  app?.unmount()
  root?.remove()
  vi.clearAllMocks()
})

const flush = async () => {
  for (let i = 0; i < 5; i++) await nextTick()
  await new Promise((r) => setTimeout(r))
}

const mount = async () => {
  store.value = reactive({ current: { uuid: 'conv-1' } })
  root = document.createElement('div')
  document.body.appendChild(root)
  app = createApp({
    render: () =>
      h(Accordion, { type: 'multiple', defaultValue: ['linear_issues'] }, () =>
        h(ConversationSideBarLinear)
      )
  })
  app.mount(root)
  await flush()
}

const issue = {
  id: 1,
  identifier: 'ENG-123',
  title: 'Fix export',
  state: 'In Progress',
  url: 'https://linear.app/clyr/issue/ENG-123'
}

describe('ConversationSideBarLinear', () => {
  it('renders nothing when Linear is not set up', async () => {
    api.getLinearIssues.mockResolvedValue({ data: { data: { enabled: false, issues: [] } } })
    await mount()
    expect(root.querySelector('[data-cy="sidebar-linear"]')).toBeNull()
  })

  it('shows linked issues with title and state', async () => {
    api.getLinearIssues.mockResolvedValue({ data: { data: { enabled: true, issues: [issue] } } })
    await mount()
    expect(api.getLinearIssues).toHaveBeenCalledWith('conv-1')
    const row = root.querySelector('[data-cy="linear-issue"]')
    expect(row.textContent).toContain('ENG-123')
    expect(row.textContent).toContain('Fix export')
    expect(row.textContent).toContain('In Progress')
    expect(row.querySelector('a').getAttribute('href')).toBe(issue.url)
  })

  it('links and unlinks an issue', async () => {
    api.getLinearIssues.mockResolvedValue({ data: { data: { enabled: true, issues: [] } } })
    api.linkLinearIssue.mockResolvedValue({ data: { data: issue } })
    api.unlinkLinearIssue.mockResolvedValue({ data: { data: true } })
    await mount()

    const input = root.querySelector('[data-cy="linear-ref"]')
    input.value = ' eng-123 '
    input.dispatchEvent(new Event('input'))
    await flush()
    root.querySelector('form').dispatchEvent(new Event('submit'))
    await flush()
    expect(api.linkLinearIssue).toHaveBeenCalledWith('conv-1', 'eng-123')
    expect(root.querySelectorAll('[data-cy="linear-issue"]').length).toBe(1)

    root.querySelector('[data-cy="linear-issue"] button').click()
    await flush()
    expect(api.unlinkLinearIssue).toHaveBeenCalledWith('conv-1', 1)
    expect(root.querySelectorAll('[data-cy="linear-issue"]').length).toBe(0)
  })

  it('ignores a failed load of a conversation the agent already left', async () => {
    let rejectFirst
    api.getLinearIssues
      .mockReturnValueOnce(new Promise((_, reject) => (rejectFirst = reject)))
      .mockResolvedValueOnce({ data: { data: { enabled: true, issues: [issue] } } })
    await mount()
    store.value.current = { uuid: 'conv-2' }
    await flush()
    rejectFirst(new Error('network'))
    await flush()
    expect(root.querySelector('[data-cy="sidebar-linear"]')).not.toBeNull()
  })

  it('hides the add and remove controls without the permission', async () => {
    can.value = false
    api.getLinearIssues.mockResolvedValue({ data: { data: { enabled: true, issues: [issue] } } })
    await mount()
    can.value = true
    expect(root.querySelector('form')).toBeNull()
    expect(root.querySelector('[data-cy="linear-issue"] button')).toBeNull()
  })
})
