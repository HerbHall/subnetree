import { useEffect } from 'react'

export interface Shortcut {
  key: string // e.g. '?', 'k', 'r', 'Escape'
  ctrl?: boolean // require Ctrl (or Cmd on Mac)
  shift?: boolean // require Shift
  handler: () => void
  description: string // for help dialog
}

/**
 * Registers global keyboard shortcuts on `document`.
 * Ignores events when the user is typing in an input, textarea, or
 * contenteditable element. Cleans up on unmount.
 */
export function useKeyboardShortcuts(shortcuts: Shortcut[]): void {
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      // Skip when typing in form elements
      const target = e.target as HTMLElement
      if (
        target instanceof HTMLInputElement ||
        target instanceof HTMLTextAreaElement ||
        target.isContentEditable
      ) {
        return
      }

      for (const shortcut of shortcuts) {
        const wantCtrl = shortcut.ctrl ?? false
        const wantShift = shortcut.shift ?? false

        // Check modifier keys -- ctrlKey or metaKey for cross-platform
        const ctrlMatch = wantCtrl ? e.ctrlKey || e.metaKey : !e.ctrlKey && !e.metaKey
        const shiftMatch = wantShift ? e.shiftKey : !e.shiftKey

        if (e.key === shortcut.key && ctrlMatch && shiftMatch) {
          e.preventDefault()
          shortcut.handler()
          return
        }
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [shortcuts])
}
