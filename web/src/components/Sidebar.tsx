import { NavLink } from 'react-router-dom'

const links = [
  { to: '/', label: 'Dashboard', icon: '⌂' },
  { to: '/digests', label: 'Digests', icon: '◉' },
  { to: '/search', label: 'Search', icon: '⌕' },
  { to: '/media', label: 'Media', icon: '⧉' },
  { to: '/urls', label: 'Links', icon: '⟁' },
]

export default function Sidebar() {
  return (
    <aside className="w-56 bg-apple-sidebar border-r border-apple-border flex flex-col shrink-0">
      <div className="px-5 py-6">
        <h1 className="text-lg font-semibold tracking-tight text-apple-text">
          Signal Sideband
        </h1>
        <p className="text-xs text-apple-secondary mt-0.5">Newsletter Platform</p>
      </div>
      <nav className="flex-1 px-3">
        {links.map(link => (
          <NavLink
            key={link.to}
            to={link.to}
            end={link.to === '/'}
            className={({ isActive }) =>
              `flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm transition-colors mb-0.5 ${
                isActive
                  ? 'bg-apple-blue text-white font-medium'
                  : 'text-apple-text hover:bg-black/5'
              }`
            }
          >
            <span className="text-base leading-none">{link.icon}</span>
            {link.label}
          </NavLink>
        ))}
      </nav>
    </aside>
  )
}
