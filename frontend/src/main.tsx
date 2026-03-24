import { render } from 'preact'
import { App } from './app.tsx'
import './index.css'

// Initialize theme on app start
const savedTheme = localStorage.getItem('theme-storage')
if (savedTheme) {
  try {
    const { state } = JSON.parse(savedTheme)
    const theme = state?.theme || 'light'
    document.documentElement.setAttribute('data-theme', theme)
    if (theme === 'dark') {
      document.documentElement.classList.add('dark')
    }
  } catch (e) {
    // Ignore parse errors
  }
}

render(<App />, document.getElementById('app')!)
