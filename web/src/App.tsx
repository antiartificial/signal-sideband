import { useState, useEffect } from 'react'
import { Routes, Route } from 'react-router-dom'
import { isAuthenticated, getAuthStatus } from './lib/api.ts'
import AppShell from './components/AppShell.tsx'
import Cerebro from './pages/Cerebro.tsx'
import Dashboard from './pages/Dashboard.tsx'
import Digests from './pages/Digests.tsx'
import DigestView from './pages/DigestView.tsx'
import Login from './pages/Login.tsx'
import Search from './pages/Search.tsx'
import MediaGallery from './pages/MediaGallery.tsx'
import URLCollection from './pages/URLCollection.tsx'

export default function App() {
  const [authState, setAuthState] = useState<'loading' | 'required' | 'ok'>('loading')

  useEffect(() => {
    getAuthStatus().then(({ required }) => {
      if (!required || isAuthenticated()) {
        setAuthState('ok')
      } else {
        setAuthState('required')
      }
    }).catch(() => {
      // If we can't reach the server, assume auth not required
      setAuthState('ok')
    })
  }, [])

  if (authState === 'loading') return null

  if (authState === 'required') {
    return <Login onLogin={() => setAuthState('ok')} />
  }

  return (
    <AppShell>
      <Routes>
        <Route path="/" element={<Dashboard />} />
        <Route path="/cerebro" element={<Cerebro />} />
        <Route path="/digests" element={<Digests />} />
        <Route path="/digests/:id" element={<DigestView />} />
        <Route path="/search" element={<Search />} />
        <Route path="/media" element={<MediaGallery />} />
        <Route path="/urls" element={<URLCollection />} />
      </Routes>
    </AppShell>
  )
}
