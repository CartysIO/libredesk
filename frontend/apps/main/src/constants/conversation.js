export const CONVERSATION_LIST_TYPE = {
  ASSIGNED: 'assigned',
  UNASSIGNED: 'unassigned',
  TEAM_UNASSIGNED: 'team_unassigned',
  VIEW: 'view',
  ALL: 'all',
  MENTIONED: 'mentioned'
}

export const CONVERSATION_DEFAULT_STATUSES = {
  OPEN: 'Open',
  SNOOZED: 'Snoozed',
  RESOLVED: 'Resolved'
}

// Kanban board statuses. "On you" and "On customer" are added by the backend.
export const CONVERSATION_BOARD_STATUSES = {
  ON_YOU: 'On you',
  ON_CUSTOMER: 'On customer'
}

// The backend treats the board statuses as default (protected) statuses too.
export const CONVERSATION_DEFAULT_STATUSES_LIST = [
  ...Object.values(CONVERSATION_DEFAULT_STATUSES),
  ...Object.values(CONVERSATION_BOARD_STATUSES)
]

// List-only status choice: not a real status. It lists every active ticket. The value is an identifier,
// not the label, so a custom status named "Active" stays an exact-name filter.
export const CONVERSATION_ACTIVE_STATUS = '__active__'
export const CONVERSATION_ACTIVE_STATUS_NAMES = [
  CONVERSATION_DEFAULT_STATUSES.OPEN,
  CONVERSATION_BOARD_STATUSES.ON_YOU,
  CONVERSATION_BOARD_STATUSES.ON_CUSTOMER
]

// Board columns in display order. `status` is the conversation status name each column lists.
// Class strings are written out in full so Tailwind keeps them.
export const CONVERSATION_BOARD_COLUMNS = [
  {
    key: 'new',
    status: CONVERSATION_DEFAULT_STATUSES.OPEN,
    labelKey: 'kanban.columns.new',
    icon: 'CircleDot',
    textClass: 'text-yellow-600 dark:text-yellow-400',
    bgClass: 'bg-yellow-500/10',
    dotClass: 'bg-yellow-500'
  },
  {
    key: 'onYou',
    status: CONVERSATION_BOARD_STATUSES.ON_YOU,
    labelKey: 'kanban.columns.onYou',
    icon: 'UserRound',
    textClass: 'text-orange-600 dark:text-orange-400',
    bgClass: 'bg-orange-500/10',
    dotClass: 'bg-orange-500'
  },
  {
    key: 'onCustomer',
    status: CONVERSATION_BOARD_STATUSES.ON_CUSTOMER,
    labelKey: 'kanban.columns.onCustomer',
    icon: 'Hourglass',
    textClass: 'text-blue-600 dark:text-blue-400',
    bgClass: 'bg-blue-500/10',
    dotClass: 'bg-blue-500'
  },
  {
    key: 'onHold',
    status: CONVERSATION_DEFAULT_STATUSES.SNOOZED,
    labelKey: 'kanban.columns.onHold',
    icon: 'CirclePause',
    textClass: 'text-purple-600 dark:text-purple-400',
    bgClass: 'bg-purple-500/10',
    dotClass: 'bg-purple-500'
  },
  {
    key: 'resolved',
    status: CONVERSATION_DEFAULT_STATUSES.RESOLVED,
    labelKey: 'kanban.columns.resolved',
    icon: 'CircleCheck',
    textClass: 'text-green-600 dark:text-green-400',
    bgClass: 'bg-green-500/10',
    dotClass: 'bg-green-500'
  }
]

export const CONVERSATION_BOARD_STATUS_LIST = CONVERSATION_BOARD_COLUMNS.map((c) => c.status)

export const MACRO_CONTEXT = {
  REPLY: 'reply',
  NEW_CONVERSATION: 'new-conversation'
}

export const TAG_ACTION = {
  ADD: 'add_tags',
  SET: 'set_tags',
  REMOVE: 'remove_tags'
}