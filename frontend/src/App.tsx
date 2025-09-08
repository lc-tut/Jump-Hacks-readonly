import { useEffect, useState } from 'react'
import './App.css'

function App() {
  const [apiHealth, setApiHealth] = useState<string>('checking...')

  useEffect(() => {
    fetch('/health')
      .then((res) => res.json())
      .then((data) => setApiHealth(data.status || 'unknown'))
      .catch(() => setApiHealth('unreachable'))
  }, [])

  return (
    <div style={{ padding: 24, fontFamily: 'system-ui, -apple-system, Segoe UI, Roboto, Helvetica, Arial, sans-serif' }}>
      <h1>Digi-Con Hackathon Frontend</h1>
      <p>API health: <strong>{apiHealth}</strong></p>
      <p>Try calling any API under <code>/api/v1</code>; the dev server proxies to <code>http://localhost:8080</code>.</p>
    </div>
  )
}

export default App
