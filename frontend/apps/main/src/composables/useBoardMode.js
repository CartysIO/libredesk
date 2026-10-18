import { ref } from 'vue'
import { readBoardMode, writeBoardMode } from '@main/utils/conversationBoard'

// Shared by every inbox list route. The choice is remembered per browser.
const isBoard = ref(readBoardMode())

export function useBoardMode() {
  const setBoardMode = (value) => {
    isBoard.value = !!value
    writeBoardMode(isBoard.value)
  }
  return { isBoard, setBoardMode }
}
