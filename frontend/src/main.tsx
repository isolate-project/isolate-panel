import { render } from 'preact'
import { App } from './app.tsx'
import './index.css'

// Initialize theme on app start
let theme = 'dark'
try {
  const savedTheme = localStorage.getItem('theme-storage')
  if (savedTheme) {
    theme = JSON.parse(savedTheme)?.state?.theme ?? 'dark'
  }
} catch {
  // corrupted localStorage — use default
}
document.documentElement.setAttribute('data-theme', theme)

render(<App />, document.getElementById('app')!)
