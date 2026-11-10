import { computed } from 'vue'
import { defineStore } from 'pinia'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { createRemoteLookup } from '@/utils/remote-lookup'
import api from '@/api'

// fetchOrganizationRows adapts the organizations API to the remote lookup: it returns
// `{ data: { data: rows } }`. The API has no `ids` filter, so ids are fetched one by one.
export async function fetchOrganizationRows({ ids, ...params }) {
  if (ids) {
    const rows = await Promise.all(
      String(ids)
        .split(',')
        .map((id) =>
          api
            .getOrganization(id)
            .then((response) => response.data.data)
            .catch(() => null)
        )
    )
    return { data: { data: rows.filter(Boolean) } }
  }
  const response = await api.getOrganizations(params)
  return { data: { data: response.data.data.results || [] } }
}

export const toOrganizationOptions = (organizations) =>
  organizations.map((org) => ({
    label: org.external_id ? `${org.name} (${org.external_id})` : org.name,
    value: String(org.id)
  }))

export const useOrganizationStore = defineStore('organization', () => {
  const emitter = useEmitter()
  const lookup = createRemoteLookup({
    fetchRows: fetchOrganizationRows,
    onError: (error) => {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  })

  const options = computed(() => toOrganizationOptions(lookup.rows.value))
  const searchOrganizations = async (query) => toOrganizationOptions(await lookup.search(query))

  return {
    organizations: lookup.rows,
    options,
    fetchOrganizations: lookup.fetchFirstPage,
    searchOrganizations,
    ensureOrganizationIDs: lookup.ensureIDs
  }
})
