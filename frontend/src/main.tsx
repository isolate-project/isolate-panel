import { render } from 'preact'
import { App } from './app.tsx'
import './index.css'

// Initialize theme on app start
const savedTheme = localStorage.getItem('theme-storage')
const theme = savedTheme ? JSON.parse(savedTheme)?.state?.theme : 'dark'
document.documentElement.setAttribute('data-theme', theme)

render(<App />, document.getElementById('app')!)
