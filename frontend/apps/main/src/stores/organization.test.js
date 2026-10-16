import { describe, it, expect, vi, beforeEach } from 'vitest'

const getOrganizations = vi.fn()
const getOrganization = vi.fn()

vi.mock('@/api', () => ({ default: { getOrganizations, getOrganization } }))
vi.mock('@/composables/useEmitter', () => ({ useEmitter: () => ({ emit: vi.fn() }) }))

const { fetchOrganizationRows, toOrganizationOptions } = await import('./organization')
const { organizationConversationFilters } = await import('@/features/organization/filters.js')

describe('fetchOrganizationRows', () => {
  beforeEach(() => {
    getOrganizations.mockReset()
    getOrganization.mockReset()
  })

  it('returns the results of a page as rows', async () => {
    getOrganizations.mockResolvedValue({
      data: { data: { results: [{ id: 1, name: 'Acme' }], total: 1 } }
    })
    const res = await fetchOrganizationRows({ page: 1, page_size: 50, q: 'ac' })
    expect(getOrganizations).toHaveBeenCalledWith({ page: 1, page_size: 50, q: 'ac' })
    expect(res.data.data).toEqual([{ id: 1, name: 'Acme' }])
  })

  it('fetches ids one by one and drops the ones that fail', async () => {
    getOrganization.mockImplementation((id) =>
      id === '2'
        ? Promise.reject(new Error('not found'))
        : Promise.resolve({ data: { data: { id: Number(id), name: `Org ${id}` } } })
    )
    const res = await fetchOrganizationRows({ ids: '1,2,3' })
    expect(getOrganizations).not.toHaveBeenCalled()
    expect(res.data.data.map((o) => o.id)).toEqual([1, 3])
  })
})

describe('toOrganizationOptions', () => {
  it('uses string ids and shows the external id', () => {
    expect(
      toOrganizationOptions([
        { id: 7, name: 'Acme', external_id: 'team_42' },
        { id: 8, name: 'Beta', external_id: null }
      ])
    ).toEqual([
      { label: 'Acme (team_42)', value: '7' },
      { label: 'Beta', value: '8' }
    ])
  })
})

describe('organizationConversationFilters', () => {
  it('filters conversations on organization_id', () => {
    expect(organizationConversationFilters(5)).toEqual([
      { model: 'conversations', field: 'organization_id', operator: 'equals', value: '5' }
    ])
  })
})
