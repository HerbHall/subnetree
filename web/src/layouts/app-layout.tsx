import { Outlet, NavLink } from 'react-router-dom'
import { useAuthStore } from '@/stores/auth'
import { Button } from '@/components/ui/button'
import { LayoutDashboard, Monitor, Settings, Info, LogOut } from 'lucide-react'

const navItems = [
  { to: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { to: '/devices', label: 'Devices', icon: Monitor },
  { to: '/settings', label: 'Settings', icon: Settings },
  { to: '/about', label: 'About', icon: Info },
]

export function AppLayout() {
  const logout = useAuthStore((s) => s.logout)
  const user = useAuthStore((s) => s.user)

  return (
    <div className="flex min-h-screen">
      <aside className="hidden w-64 flex-col border-r border-[var(--nv-border-subtle)] bg-[var(--nv-sidebar-bg)] md:flex">
        <div className="border-b border-[var(--nv-border-subtle)] p-4">
          <h1 className="text-lg font-semibold text-[var(--nv-green-400)]">NetVantage</h1>
        </div>
        <nav className="flex-1 space-y-1 p-3" aria-label="Main navigation">
          {navItems.map(({ to, label, icon: Icon }) => (
            <NavLink
              key={to}
              to={to}
              className={({ isActive }) =>
                `flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors ${
                  isActive
                    ? 'bg-[var(--nv-sidebar-active-bg)] text-[var(--nv-sidebar-active)]'
                    : 'text-[var(--nv-sidebar-item)] hover:bg-[var(--nv-bg-hover)]'
                }`
              }
            >
              <Icon className="h-4 w-4" />
              {label}
            </NavLink>
          ))}
        </nav>
        <div className="border-t border-[var(--nv-border-subtle)] p-3">
          <div className="mb-2 px-3 text-xs text-[var(--nv-text-muted)]">
            {user?.username ?? 'User'}
          </div>
          <Button
            variant="ghost"
            size="sm"
            className="w-full justify-start gap-3 text-[var(--nv-sidebar-item)]"
            onClick={logout}
          >
            <LogOut className="h-4 w-4" />
            Sign out
          </Button>
        </div>
      </aside>
      <main className="flex-1 p-6">
        <Outlet />
      </main>
    </div>
  )
}
