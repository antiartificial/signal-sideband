import type { ReactNode } from 'react'
import Sidebar from './Sidebar.tsx'
import MobileNav from './MobileNav.tsx'

export default function AppShell({ children }: { children: ReactNode }) {
  return (
    <div className="flex h-screen bg-apple-bg transition-colors duration-300">
      {/* Desktop sidebar */}
      <div className="hidden md:block">
        <Sidebar />
      </div>
      <main className="flex-1 overflow-y-auto pb-20 md:pb-0">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 py-6 sm:py-8 animate-fade-in">
          {children}
        </div>
      </main>
      {/* Mobile bottom nav */}
      <div className="md:hidden">
        <MobileNav />
      </div>
    </div>
  )
}
