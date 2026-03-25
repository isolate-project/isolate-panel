import { useEffect, useCallback } from 'preact/hooks';

export interface KeyboardShortcut {
  key: string;
  ctrl?: boolean;
  shift?: boolean;
  alt?: boolean;
  action: () => void;
  description?: string;
  preventDefault?: boolean;
}

export interface UseKeyboardShortcutsOptions {
  shortcuts: KeyboardShortcut[];
  enabled?: boolean;
}

/**
 * Hook for managing keyboard shortcuts
 * @param options - Keyboard shortcut options
 */
export function useKeyboardShortcuts({
  shortcuts,
  enabled = true,
}: UseKeyboardShortcutsOptions) {
  const handleKeyDown = useCallback(
    (event: KeyboardEvent) => {
      if (!enabled) return;

      const { key, ctrlKey, shiftKey, altKey } = event;

      for (const shortcut of shortcuts) {
        const matches =
          key.toLowerCase() === shortcut.key.toLowerCase() &&
          (shortcut.ctrl === undefined || ctrlKey === shortcut.ctrl) &&
          (shortcut.shift === undefined || shiftKey === shortcut.shift) &&
          (shortcut.alt === undefined || altKey === shortcut.alt);

        if (matches) {
          if (shortcut.preventDefault !== false) {
            event.preventDefault();
          }
          shortcut.action();
          break;
        }
      }
    },
    [shortcuts, enabled]
  );

  useEffect(() => {
    if (!enabled) return;

    window.addEventListener('keydown', handleKeyDown);

    return () => {
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [handleKeyDown, enabled]);
}

/**
 * Get all registered shortcuts as a formatted list
 * @param shortcuts - Array of keyboard shortcuts
 * @returns Formatted list of shortcuts for display
 */
export function formatShortcuts(shortcuts: KeyboardShortcut[]): string[] {
  return shortcuts
    .filter((s) => s.description)
    .map((s) => {
      const modifiers = [];
      if (s.ctrl) modifiers.push('Ctrl');
      if (s.alt) modifiers.push('Alt');
      if (s.shift) modifiers.push('Shift');
      
      const key = s.key.toUpperCase();
      const shortcut = [...modifiers, key].join('+');
      
      return `${shortcut} - ${s.description}`;
    });
}

/**
 * Common keyboard shortcuts for the application
 */
export const commonShortcuts: Record<string, KeyboardShortcut> = {
  save: {
    key: 's',
    ctrl: true,
    action: () => {},
    description: 'Save current form',
  },
  close: {
    key: 'escape',
    action: () => {},
    description: 'Close modal/dialog',
  },
  search: {
    key: 'k',
    ctrl: true,
    action: () => {},
    description: 'Quick search',
  },
  help: {
    key: '?',
    action: () => {},
    description: 'Show keyboard shortcuts help',
  },
  refresh: {
    key: 'r',
    ctrl: true,
    action: () => {},
    description: 'Refresh page',
  },
};
