import { CONVERSATION_BOARD_COLUMNS } from '@main/constants/conversation'

// A custom status that has no board column reads as "on hold": nobody has a move defined for it.
const FALLBACK_CODE = { key: 'other', textClass: 'text-code-hold', hollow: false }

// The desk's whose-move code for a conversation status: the colour of its board column.
export const statusCode = (status) =>
  CONVERSATION_BOARD_COLUMNS.find((column) => column.status === status) || FALLBACK_CODE
