import {
  CONVERSATION_ACTIVE_STATUS,
  CONVERSATION_ACTIVE_STATUS_NAMES
} from '@main/constants/conversation'

// Filter rule for the list status choice. "Active" is a list-only choice, so it
// becomes an "in" rule on the status names. Any other status is an exact match.
export const listStatusFilter = (status) => {
  if (status === CONVERSATION_ACTIVE_STATUS) {
    return {
      model: 'conversation_statuses',
      field: 'name',
      operator: 'in',
      value: JSON.stringify(CONVERSATION_ACTIVE_STATUS_NAMES)
    }
  }
  return { model: 'conversation_statuses', field: 'name', operator: 'equals', value: status }
}

export const isValidListStatus = (status, statuses) =>
  status === CONVERSATION_ACTIVE_STATUS || statuses.some((s) => s.name === status)

// True when a conversation status belongs to the list status choice. An empty choice matches all.
export const matchesListStatus = (conversationStatus, listStatus) => {
  if (!listStatus) return true
  if (listStatus === CONVERSATION_ACTIVE_STATUS) {
    return CONVERSATION_ACTIVE_STATUS_NAMES.includes(conversationStatus)
  }
  return conversationStatus === listStatus
}
