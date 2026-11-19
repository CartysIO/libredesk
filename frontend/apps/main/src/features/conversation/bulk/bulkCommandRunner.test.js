// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  BULK_UNDO_WINDOW_MS,
  bulkCommandPending,
  flushActiveBulkCommand,
  runDeferredBulkCommand
} from './bulkCommandRunner'

vi.mock('vue-sonner', () => ({
  toast: vi.fn()
}))

describe('bulkCommandRunner', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(async () => {
    await flushActiveBulkCommand()
    vi.useRealTimers()
    vi.clearAllMocks()
  })

  it('applies immediately and commits after the undo window', async () => {
    const apply = vi.fn()
    const revert = vi.fn()
    const commit = vi.fn().mockResolvedValue(undefined)

    runDeferredBulkCommand(
      { apply, revert, commit, toastDescription: 'pending' },
      { undoLabel: 'Undo' }
    )
    expect(apply).toHaveBeenCalledTimes(1)
    expect(bulkCommandPending.value).toBe(true)
    expect(commit).not.toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(BULK_UNDO_WINDOW_MS)
    expect(bulkCommandPending.value).toBe(false)
    expect(commit).toHaveBeenCalledTimes(1)
    expect(revert).not.toHaveBeenCalled()
  })

  it('blocks a second command while the undo window is open', () => {
    const apply = vi.fn()
    const revert = vi.fn()
    const commit = vi.fn().mockResolvedValue(undefined)
    const onBlocked = vi.fn()

    runDeferredBulkCommand(
      { apply, revert, commit, toastDescription: 'first' },
      { undoLabel: 'Undo' }
    )
    const started = runDeferredBulkCommand(
      { apply, revert, commit, toastDescription: 'second' },
      { undoLabel: 'Undo', onBlocked }
    )

    expect(started).toBe(false)
    expect(onBlocked).toHaveBeenCalledTimes(1)
    expect(apply).toHaveBeenCalledTimes(1)
  })

  it('reverts when undo is triggered before commit', async () => {
    const apply = vi.fn()
    const revert = vi.fn()
    const commit = vi.fn().mockResolvedValue(undefined)
    const { toast } = await import('vue-sonner')

    runDeferredBulkCommand(
      { apply, revert, commit, toastDescription: 'pending' },
      { undoLabel: 'Undo' }
    )

    const action = toast.mock.calls[0][1].action
    action.onClick()

    await vi.advanceTimersByTimeAsync(BULK_UNDO_WINDOW_MS)
    expect(revert).toHaveBeenCalledTimes(1)
    expect(commit).not.toHaveBeenCalled()
    expect(bulkCommandPending.value).toBe(false)
  })
})