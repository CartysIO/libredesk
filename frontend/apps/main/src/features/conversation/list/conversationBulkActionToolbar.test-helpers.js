import { nextTick } from 'vue'

export async function settleBulkToolbar (ticks = 10) {
  for (let i = 0; i < ticks; i++) await nextTick()
}

/** Last button in the bulk delete/spam alert dialog (first is Cancel). */
export function bulkToolbarConfirmButton () {
  const buttons = document.querySelectorAll('[role="alertdialog"] button')
  return buttons[buttons.length - 1]
}

/** Radix AlertDialogAction closes on pointerdown; mouse confirm must use pointerdown. */
export async function confirmBulkToolbarAction (btn = bulkToolbarConfirmButton()) {
  btn.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true, cancelable: true }))
  await settleBulkToolbar()
}

/** Keyboard activation (Enter / Space on the focused destructive action). */
export async function confirmBulkToolbarActionKeyboard (
  key,
  btn = bulkToolbarConfirmButton()
) {
  btn.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }))
  await settleBulkToolbar()
}