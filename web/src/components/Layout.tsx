import { Link, Outlet, useLocation } from 'react-router-dom'

const navLinks = [
  { to: '/', label: '首页' },
  { to: '/languages', label: '语言' },
  { to: '/settings', label: '设置' },
]

export default function Layout() {
  const { pathname } = useLocation()

  return (
    <div className="min-h-screen bg-stone-950 text-stone-100">
      <header className="sticky top-0 z-50 border-b border-stone-800/80 bg-stone-950/80 backdrop-blur-md">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-6 py-4">
          <Link to="/" className="text-xl font-bold tracking-tight">
            <span className="text-amber-500">Learn</span>
            <span className="text-stone-200">Code</span>
          </Link>
          <nav className="flex items-center gap-1">
            {navLinks.map(({ to, label }) => {
              const active = pathname === to || (to !== '/' && pathname.startsWith(to))
              return (
                <Link
                  key={to}
                  to={to}
                  className={`rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
                    active
                      ? 'text-amber-500'
                      : 'text-stone-400 hover:text-stone-200'
                  }`}
                >
                  {label}
                </Link>
              )
            })}
          </nav>
        </div>
      </header>
      <main className="mx-auto max-w-6xl px-6 py-10">
        <Outlet />
      </main>
    </div>
  )
}
