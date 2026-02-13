import { NavLink } from 'react-router-dom'
import { useTheme } from '../lib/theme.ts'

const links = [
  { to: '/', label: 'Dashboard', icon: 'fa-house' },
  { to: '/cerebro', label: 'Cerebro', icon: 'fa-circle-nodes' },
  { to: '/digests', label: 'Digests', icon: 'fa-newspaper' },
  { to: '/search', label: 'Search', icon: 'fa-magnifying-glass' },
  { to: '/media', label: 'Media', icon: 'fa-images' },
  { to: '/urls', label: 'Links', icon: 'fa-link' },
]

export default function Sidebar() {
  const { theme, toggleTheme } = useTheme()

  return (
    <aside className="w-56 bg-apple-sidebar border-r border-apple-border flex flex-col shrink-0 transition-colors duration-300">
      <div className="px-5 py-6 animate-fade-in">
        <h1 className="text-lg font-semibold tracking-tight text-apple-text">
          Signal Sideband
        </h1>
        <p className="text-xs text-apple-secondary mt-0.5 font-mono tracking-wide">
          // signal intelligence
        </p>
      </div>
      <nav className="flex-1 px-3">
        {links.map((link, i) => (
          <NavLink
            key={link.to}
            to={link.to}
            end={link.to === '/'}
            className={({ isActive }) =>
              `flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm mb-0.5 animate-slide-in
              transition-all duration-200 ease-out ${
                isActive
                  ? 'bg-apple-blue text-white font-medium shadow-sm'
                  : 'text-apple-text hover:bg-apple-accent-dim hover:translate-x-1'
              }`
            }
            style={{ animationDelay: `${i * 50}ms` }}
          >
            <i className={`fawsb ${link.icon} text-sm w-4 text-center`} />
            {link.label}
          </NavLink>
        ))}
      </nav>
      <div className="px-3 pb-4">
        <button
          onClick={toggleTheme}
          className="flex items-center gap-2.5 w-full px-3 py-2 rounded-lg text-sm text-apple-secondary
            hover:bg-apple-accent-dim hover:text-apple-text transition-all duration-200"
        >
          <i className={`fawsb ${theme === 'dark' ? 'fa-sun' : 'fa-moon'} text-sm w-4 text-center`} />
          {theme === 'dark' ? 'Light Mode' : 'Dark Mode'}
        </button>
      </div>
    </aside>
  )
}
