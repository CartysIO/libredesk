import { ref } from 'vue'
import { toast } from 'vue-sonner'

/** Grace period before a bulk command commits to the server. */
export const BULK_UNDO_WINDOW_MS = 8000

/** True while a deferred bulk command is in its undo window (reactive for toolbar visibility). */
export const bulkCommandPending = ref(false)

/**
 * @typedef {object} BulkCommand
 * @property {() => void} apply Optimistic UI update (sync).
 * @property {() => void} revert Undo optimistic update (sync).
 * @property {() => Promise<void>} commit Persist to the server.
 * @property {string} toastDescription Shown while the undo window is open.
 */

let activeCommand = null
let commitTimer = null

const clearActiveCommand = () => {
  activeCommand = null
  bulkCommandPending.value = false
}

export const hasActiveBulkCommand = () => activeCommand !== null

const clearCommitTimer = () => {
  if (commitTimer !== null) {
    clearTimeout(commitTimer)
    commitTimer = null
  }
}

/** Commits any in-flight deferred bulk command immediately. */
export async function flushActiveBulkCommand () {
  clearCommitTimer()
  const cmd = activeCommand
  clearActiveCommand()
  if (!cmd) return
  await cmd.commit()
}

const scheduleCommit = (command) => {
  clearCommitTimer()
  commitTimer = setTimeout(async () => {
    commitTimer = null
    if (activeCommand !== command) return
    clearActiveCommand()
    try {
      await command.commit()
    } catch {
      command.revert()
    }
  }, BULK_UNDO_WINDOW_MS)
}

/**
 * Runs a bulk command: optimistic apply, undo toast, deferred commit.
 * @param {BulkCommand} command
 * @param {{ undoLabel: string, onBlocked?: () => void }} options
 * @returns {boolean} False when another command is still in its undo window.
 */
export function runDeferredBulkCommand (command, { undoLabel, onBlocked }) {
  if (activeCommand) {
    onBlocked?.()
    return false
  }

  activeCommand = command
  bulkCommandPending.value = true
  command.apply()
  scheduleCommit(command)

  toast(command.toastDescription, {
    duration: BULK_UNDO_WINDOW_MS,
    action: {
      label: undoLabel,
      onClick: () => {
        if (activeCommand !== command) return
        clearCommitTimer()
        clearActiveCommand()
        command.revert()
      }
    }
  })

  return true
}