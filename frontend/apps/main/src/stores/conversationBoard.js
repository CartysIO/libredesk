import { defineStore } from 'pinia'
import { reactive, ref, watch } from 'vue'
import { useThrottleFn } from '@vueuse/core'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { deepMerge } from '@shared-ui/utils/object.js'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import { CONVERSATION_LIST_TYPE, CONVERSATION_BOARD_STATUS_LIST } from '@main/constants/conversation'
import { subscribeListReplace } from '@main/websocket'
import { useConversationStore } from '@main/stores/conversation'
import { useUserStore } from '@main/stores/user'
import {
  BOARD_PAGE_SIZE,
  applyBoardMove,
  boardStatusFilter,
  conversationMatchesScope,
  createBoardColumns,
  findBoardCard,
  groupByBoardStatus,
  undoBoardMove
} from '@main/utils/conversationBoard'
import api from '@main/api'

// A view cannot be filtered by status on the server, so its pages are split into columns here.
const VIEW_INITIAL_PAGES = 4
const REFRESH_THROTTLE_MS = 5000
// A refresh reloads every page the column holds, up to this many.
const REFRESH_MAX_PAGES = 8

export const useConversationBoardStore = defineStore('conversationBoard', () => {
  const conversationStore = useConversationStore()
  const userStore = useUserStore()
  const emitter = useEmitter()

  const columns = reactive(createBoardColumns())
  const context = reactive({ listType: null, teamID: 0, viewID: 0, active: false, ready: false })
  const view = reactive({ page: 0, hasMore: false, fetching: false })
  const dragging = ref(null)
  // Bumped when the list context changes so late responses are dropped.
  let seq = 0

  const isView = () => context.listType === CONVERSATION_LIST_TYPE.VIEW

  const toast = (error) =>
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })

  function requestPage(page, status) {
    const params = {
      page,
      page_size: BOARD_PAGE_SIZE,
      ...conversationStore.getListSortParams()
    }
    const withFilters = { ...params, filters: JSON.stringify(boardStatusFilter(status)) }
    switch (context.listType) {
      case CONVERSATION_LIST_TYPE.ASSIGNED:
        return api.getAssignedConversations(withFilters)
      case CONVERSATION_LIST_TYPE.UNASSIGNED:
        return api.getUnassignedConversations(withFilters)
      case CONVERSATION_LIST_TYPE.ALL:
        return api.getAllConversations(withFilters)
      case CONVERSATION_LIST_TYPE.MENTIONED:
        return api.getMentionedConversations(withFilters)
      case CONVERSATION_LIST_TYPE.TEAM_UNASSIGNED:
        return api.getTeamUnassignedConversations(context.teamID, withFilters)
      case CONVERSATION_LIST_TYPE.VIEW:
        return api.getViewConversations(context.viewID, params)
      default:
        return Promise.reject(new Error('Invalid conversation list type: ' + context.listType))
    }
  }

  async function fetchColumn(status, { reset = false } = {}) {
    if (isView()) return reset ? loadView(true) : loadMoreView()
    const col = columns[status]
    if (!col || col.fetching) return
    if (!reset && !col.hasMore) return
    const page = reset ? 1 : col.page + 1
    const mySeq = seq
    col.fetching = true
    if (!col.items.length) col.loading = true
    try {
      let data
      if (reset) {
        // Reload every page the column holds, so cards deleted or moved on the server drop out.
        const pages = Math.min(Math.max(col.page, 1), REFRESH_MAX_PAGES)
        const rows = []
        let current = 0
        let hasMore = true
        while (hasMore && current < pages) {
          current += 1
          const response = await requestPage(current, status)
          if (mySeq !== seq) return
          data = response.data.data
          rows.push(...(data.results || []))
          hasMore = data.total_pages > current
        }
        const seen = new Set()
        col.items = rows.filter((c) => !seen.has(c.uuid) && seen.add(c.uuid))
        col.page = current
      } else {
        const response = await requestPage(page, status)
        if (mySeq !== seq) return
        data = response.data.data
        const known = new Set(col.items.map((c) => c.uuid))
        col.items.push(...(data.results || []).filter((c) => !known.has(c.uuid)))
        col.page = page
      }
      col.hasMore = data.total_pages > col.page
      col.total = data.total
      col.error = ''
    } catch (error) {
      if (mySeq !== seq) return
      col.error = handleHTTPError(error).message
    } finally {
      if (mySeq === seq) {
        col.fetching = false
        col.loading = false
      }
    }
  }

  function distributeView(rows, replace) {
    const groups = groupByBoardStatus(rows)
    for (const status of CONVERSATION_BOARD_STATUS_LIST) {
      const col = columns[status]
      if (replace) {
        col.items = groups[status]
      } else {
        const known = new Set(col.items.map((c) => c.uuid))
        col.items.push(...groups[status].filter((c) => !known.has(c.uuid)))
      }
      col.total = col.items.length
      col.hasMore = view.hasMore
      col.error = ''
    }
  }

  async function loadView(reset) {
    if (view.fetching) return
    const mySeq = seq
    view.fetching = true
    const hadItems = CONVERSATION_BOARD_STATUS_LIST.some((s) => columns[s].items.length)
    if (!hadItems) CONVERSATION_BOARD_STATUS_LIST.forEach((s) => (columns[s].loading = true))
    try {
      const rows = []
      let page = 0
      let hasMore = true
      while (hasMore && page < VIEW_INITIAL_PAGES) {
        page += 1
        const response = await requestPage(page)
        if (mySeq !== seq) return
        const data = response.data.data
        rows.push(...(data.results || []))
        hasMore = data.total_pages > page
      }
      view.page = page
      view.hasMore = hasMore
      distributeView(rows, reset)
    } catch (error) {
      if (mySeq !== seq) return
      CONVERSATION_BOARD_STATUS_LIST.forEach((s) => (columns[s].error = handleHTTPError(error).message))
    } finally {
      if (mySeq === seq) {
        view.fetching = false
        CONVERSATION_BOARD_STATUS_LIST.forEach((s) => (columns[s].loading = false))
      }
    }
  }

  async function loadMoreView() {
    if (view.fetching || !view.hasMore) return
    const mySeq = seq
    view.fetching = true
    CONVERSATION_BOARD_STATUS_LIST.forEach((s) => (columns[s].fetching = true))
    try {
      const response = await requestPage(view.page + 1)
      if (mySeq !== seq) return
      const data = response.data.data
      view.page += 1
      view.hasMore = data.total_pages > view.page
      distributeView(data.results || [], false)
    } catch (error) {
      if (mySeq === seq) toast(error)
    } finally {
      if (mySeq === seq) {
        view.fetching = false
        CONVERSATION_BOARD_STATUS_LIST.forEach((s) => (columns[s].fetching = false))
      }
    }
  }

  function loadAll() {
    if (isView()) return loadView(true)
    return Promise.all(CONVERSATION_BOARD_STATUS_LIST.map((s) => fetchColumn(s, { reset: true })))
  }

  function resetColumns() {
    Object.assign(columns, createBoardColumns())
    view.page = 0
    view.hasMore = false
    view.fetching = false
  }

  // Called when the board shows. Same list context: refresh in place. New context: start over.
  function activate({ listType, teamID = 0, viewID = 0 }) {
    const changed =
      context.listType !== listType ||
      String(context.teamID) !== String(teamID) ||
      String(context.viewID) !== String(viewID)
    context.active = true
    if (changed || !context.ready) {
      seq++
      resetColumns()
      context.listType = listType
      context.teamID = teamID
      context.viewID = viewID
      context.ready = true
    }
    return loadAll()
  }

  function deactivate() {
    if (!context.active) return
    context.active = false
    // Hand the live subscription back to the list.
    subscribeListReplace(conversationStore.conversations.data?.map((c) => c.uuid) || [])
  }

  const refreshAll = () => (context.active && context.ready ? loadAll() : undefined)
  const throttledRefresh = useThrottleFn(refreshAll, REFRESH_THROTTLE_MS, true)

  function removeCard(uuid) {
    const found = findBoardCard(columns, uuid)
    if (!found) return
    const col = columns[found.status]
    col.items.splice(found.index, 1)
    col.total = Math.max(0, col.total - 1)
  }

  // Drops deleted or spam conversations. The server sends no event for them.
  function removeConversations(uuids) {
    for (const uuid of uuids) removeCard(uuid)
  }

  const cardUUIDs = () =>
    context.active ? CONVERSATION_BOARD_STATUS_LIST.flatMap((s) => columns[s].items.map((c) => c.uuid)) : []

  const scopeOf = (conv) =>
    conversationMatchesScope(context.listType, conv, {
      userID: userStore.userID,
      teamID: context.teamID
    })

  // Returns true when the board handled the update.
  function mergeLive(payload, { insert }) {
    if (!context.active || !payload?.uuid) return false
    const found = findBoardCard(columns, payload.uuid)
    if (found) {
      const previous = found.status
      deepMerge(found.card, payload)
      if (payload.status && payload.status !== previous) {
        found.card.status = previous
        applyBoardMove(columns, payload.uuid, payload.status)
      }
      if (scopeOf(found.card) === false) removeCard(payload.uuid)
      return true
    }
    const col = columns[payload.status]
    if (insert && col) {
      const scope = scopeOf(payload)
      if (scope === true) {
        col.items.unshift(payload)
        col.total += 1
      } else if (scope === null) {
        throttledRefresh()
      }
      return true
    }
    // A partial update for a card we do not hold may move it into view.
    if (!insert && (payload.status || 'assigned_user_id' in payload)) throttledRefresh()
    return false
  }

  const handleConvPush = (payload) => mergeLive(payload, { insert: true })
  const mergeConversationUpdate = (update) => mergeLive(update, { insert: false })

  function mergeContactUpdate(update) {
    if (!context.active) return
    const { contact_id, ...fields } = update
    for (const status of CONVERSATION_BOARD_STATUS_LIST) {
      for (const card of columns[status].items) {
        if (card.contact_id === contact_id) {
          if (!card.contact) card.contact = {}
          deepMerge(card.contact, fields)
        }
      }
    }
  }

  function incrementUnread(uuid) {
    const found = findBoardCard(columns, uuid)
    if (!found) return
    found.card.unread_message_count = Math.min((found.card.unread_message_count || 0) + 1, 10)
  }

  function applyConversationRead(state) {
    const found = findBoardCard(columns, state.conversation_uuid)
    if (!found) return
    if (state.is_unread) {
      found.card.unread_message_count = Math.max(1, found.card.unread_message_count || 0)
    } else if (Date.parse(found.card.last_message_at) <= Date.parse(state.last_seen_at)) {
      found.card.unread_message_count = 0
    }
  }

  // Optimistic move. Reverts and shows a toast when the server refuses.
  async function moveCard(uuid, toStatus, snoozedUntil = '') {
    const move = applyBoardMove(columns, uuid, toStatus)
    if (!move) return false
    try {
      const body = { status: toStatus }
      if (snoozedUntil) body.snoozed_until = snoozedUntil
      await api.updateConversationStatus(uuid, body)
      conversationStore.refreshSidebarCounts?.()
      return true
    } catch (error) {
      // A websocket update may have moved the card meanwhile. Undo only when it is still where we put it.
      undoBoardMove(columns, move)
      toast(error)
      return false
    }
  }

  // The socket subscription must cover the cards on the board.
  watch(
    () =>
      context.active
        ? CONVERSATION_BOARD_STATUS_LIST.flatMap((s) => columns[s].items.map((c) => c.uuid))
            .sort()
            .join(',')
        : null,
    (key) => {
      if (key === null) return
      subscribeListReplace(key ? key.split(',') : [])
    }
  )

  return {
    columns,
    context,
    dragging,
    activate,
    deactivate,
    fetchColumn,
    refreshAll,
    moveCard,
    handleConvPush,
    mergeConversationUpdate,
    mergeContactUpdate,
    removeConversations,
    cardUUIDs,
    incrementUnread,
    applyConversationRead
  }
})
