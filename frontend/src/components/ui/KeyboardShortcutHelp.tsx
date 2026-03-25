import { Modal } from './Modal';
import { formatShortcuts, type KeyboardShortcut } from '../hooks/useKeyboardShortcuts';

interface KeyboardShortcutHelpProps {
  isOpen: boolean;
  onClose: () => void;
  shortcuts: KeyboardShortcut[];
}

export function KeyboardShortcutHelp({
  isOpen,
  onClose,
  shortcuts,
}: KeyboardShortcutHelpProps) {
  const formattedShortcuts = formatShortcuts(shortcuts);

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title="Keyboard Shortcuts"
      size="md"
    >
      <div className="space-y-4">
        <p className="text-sm text-gray-600 dark:text-gray-400">
          Quick access to common actions using keyboard shortcuts:
        </p>
        
        {formattedShortcuts.length > 0 ? (
          <ul className="space-y-2">
            {formattedShortcuts.map((shortcut, index) => (
              <li
                key={index}
                className="flex items-center justify-between py-2 px-3 bg-gray-50 dark:bg-gray-800 rounded-lg"
              >
                <span className="text-sm text-gray-700 dark:text-gray-300">
                  {shortcut.split(' - ')[1]}
                </span>
                <kbd className="px-2 py-1 text-xs font-mono bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded text-gray-800 dark:text-gray-200">
                  {shortcut.split(' - ')[0]}
                </kbd>
              </li>
            ))}
          </ul>
        ) : (
          <p className="text-sm text-gray-500 dark:text-gray-400">
            No keyboard shortcuts available.
          </p>
        )}
        
        <div className="mt-6 pt-4 border-t border-gray-200 dark:border-gray-700">
          <p className="text-xs text-gray-500 dark:text-gray-400">
            Tip: Press <kbd className="px-1 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 rounded">?</kbd> to open this help dialog.
          </p>
        </div>
      </div>
      
      <div className="mt-6 flex justify-end">
        <button
          onClick={onClose}
          className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
        >
          Got it
        </button>
      </div>
    </Modal>
  );
}
