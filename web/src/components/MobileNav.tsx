import { NavLink } from 'react-router-dom'
import { useTheme } from '../lib/theme.ts'

const tabs = [
  { to: '/', label: 'Home', icon: 'fa-house' },
  { to: '/cerebro', label: 'Cerebro', icon: 'fa-circle-nodes' },
  { to: '/digests', label: 'Digests', icon: 'fa-newspaper' },
  { to: '/search', label: 'Search', icon: 'fa-magnifying-glass' },
  { to: '/media', label: 'Media', icon: 'fa-images' },
  { to: '/urls', label: 'Links', icon: 'fa-link' },
  { to: '/settings', label: 'Settings', icon: 'fa-gear' },
]

export default function MobileNav() {
  const { theme, toggleTheme } = useTheme()

  return (
    <nav className="fixed bottom-0 left-0 right-0 bg-apple-sidebar/95 backdrop-blur-lg border-t border-apple-border
      safe-area-pb transition-colors duration-300 z-40">
      <div className="flex items-center justify-around px-2 pt-2 pb-1">
        {tabs.map(tab => (
          <NavLink
            key={tab.to}
            to={tab.to}
            end={tab.to === '/'}
            className={({ isActive }) =>
              `flex flex-col items-center gap-0.5 px-3 py-1.5 rounded-lg text-[10px] min-w-[40px]
              transition-all duration-200 active:scale-95 ${
                isActive
                  ? 'text-apple-blue font-semibold'
                  : 'text-apple-secondary'
              }`
            }
          >
            <i className={`fawsb ${tab.icon} text-lg`} />
            <span>{tab.label}</span>
          </NavLink>
        ))}
        <button
          onClick={toggleTheme}
          className="flex flex-col items-center gap-0.5 px-3 py-1.5 rounded-lg text-[10px] min-w-[40px]
            text-apple-secondary transition-all duration-200 active:scale-95"
        >
          <i className={`fawsb ${theme === 'dark' ? 'fa-sun' : 'fa-moon'} text-lg`} />
          <span>{theme === 'dark' ? 'Light' : 'Dark'}</span>
        </button>
      </div>
    </nav>
  )
}
