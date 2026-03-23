import { useState, useEffect } from 'preact/hooks'

export function App() {
  const [apiStatus, setApiStatus] = useState<string>('Checking...')

  useEffect(() => {
    fetch('/api/')
      .then((res) => res.json())
      .then((data) => {
        setApiStatus(`Connected: ${data.message} v${data.version}`)
      })
      .catch(() => {
        setApiStatus('API connection failed')
      })
  }, [])

  return (
    <div className="min-h-screen bg-gray-100 flex items-center justify-center">
      <div className="bg-white p-8 rounded-lg shadow-lg max-w-md w-full">
        <h1 className="text-3xl font-bold text-gray-800 mb-4">Isolate Panel</h1>
        <p className="text-gray-600 mb-4">
          Lightweight proxy core management panel
        </p>
        <div className="bg-blue-50 border border-blue-200 rounded p-4">
          <p className="text-sm text-blue-800">
            <strong>API Status:</strong> {apiStatus}
          </p>
        </div>
        <div className="mt-6 text-sm text-gray-500">
          <p>Phase 0: Setup Complete ✓</p>
          <p className="mt-2">
            Frontend: Preact + Vite + TypeScript + Tailwind CSS
          </p>
          <p>Backend: Go + Fiber + GORM + SQLite</p>
        </div>
      </div>
    </div>
  )
}
