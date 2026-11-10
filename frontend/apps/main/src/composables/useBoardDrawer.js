import { ref } from 'vue'
import { readDrawerWidth, writeDrawerWidth } from '@main/utils/conversationBoard'

// Shared drawer state. width 0 means the default (75% of the content area, at least 720px).
const expanded = ref(false)
const width = ref(readDrawerWidth())
// The drawer's rendered width in px. The board pads its scroll area by it, so every column can scroll into view.
const occupied = ref(0)

export function useBoardDrawer() {
  const toggleExpanded = () => {
    expanded.value = !expanded.value
  }
  const resetExpanded = () => {
    expanded.value = false
  }
  const setWidth = (value) => {
    width.value = value
    writeDrawerWidth(value)
  }
  return { expanded, width, occupied, toggleExpanded, resetExpanded, setWidth }
}
