// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createApp, h, nextTick } from 'vue'

const { ensureOrganizationIDs, picker } = vi.hoisted(() => ({
  ensureOrganizationIDs: vi.fn(),
  picker: { emitSelect: null }
}))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key) => key }) }))
vi.mock('@/stores/organization', () => ({
  useOrganizationStore: () => ({
    options: [
      { value: '3', label: 'Acme (team_1)' },
      { value: '12', label: 'Globex' }
    ],
    ensureOrganizationIDs
  })
}))
vi.mock('@/components/combobox/SelectOrganizationCombobox.vue', () => ({
  default: {
    emits: ['select'],
    setup(_, { emit }) {
      picker.emitSelect = (item) => emit('select', item)
      return () => null
    }
  }
}))
import SelectOrganizationsCombobox from './SelectOrganizationsCombobox.vue'

let app
let root
afterEach(() => {
  app?.unmount()
  root?.remove()
  vi.clearAllMocks()
})

const mount = async (modelValue) => {
  const updates = []
  root = document.createElement('div')
  document.body.appendChild(root)
  app = createApp({
    render: () =>
      h(SelectOrganizationsCombobox, { modelValue, 'onUpdate:modelValue': (v) => updates.push(v) })
  })
  app.mount(root)
  await nextTick()
  return updates
}

describe('SelectOrganizationsCombobox', () => {
  it('shows the selected organizations by name and loads their names', async () => {
    await mount('3,99')
    expect(root.textContent).toContain('Acme (team_1)')
    expect(root.textContent).toContain('#99')
    expect(ensureOrganizationIDs).toHaveBeenCalledWith(['3', '99'])
  })

  it('adds a picked organization once, as a comma separated list', async () => {
    const updates = await mount('3')
    picker.emitSelect({ value: 12, label: 'Globex' })
    picker.emitSelect({ value: '3', label: 'Acme' })
    expect(updates).toEqual(['3,12'])
  })

  it('removes an organization', async () => {
    const updates = await mount('3,12')
    root.querySelector('button').click()
    expect(updates).toEqual(['12'])
  })
})
