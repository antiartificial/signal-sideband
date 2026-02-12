import { Routes, Route } from 'react-router-dom'
import AppShell from './components/AppShell.tsx'
import Dashboard from './pages/Dashboard.tsx'
import Digests from './pages/Digests.tsx'
import DigestView from './pages/DigestView.tsx'
import Search from './pages/Search.tsx'
import MediaGallery from './pages/MediaGallery.tsx'
import URLCollection from './pages/URLCollection.tsx'

export default function App() {
  return (
    <AppShell>
      <Routes>
        <Route path="/" element={<Dashboard />} />
        <Route path="/digests" element={<Digests />} />
        <Route path="/digests/:id" element={<DigestView />} />
        <Route path="/search" element={<Search />} />
        <Route path="/media" element={<MediaGallery />} />
        <Route path="/urls" element={<URLCollection />} />
      </Routes>
    </AppShell>
  )
}
