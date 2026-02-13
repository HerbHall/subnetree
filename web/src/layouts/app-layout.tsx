import { useState, useCallback, useMemo } from 'react'
import { Outlet, NavLink, useNavigate } from 'react-router-dom'
import { useAuthStore } from '@/stores/auth'
import { Button } from '@/components/ui/button'
import { LayoutDashboard, Monitor, Network, Bot, Layers, FileText, Settings, Info, LogOut } from 'lucide-react'
import { useKeyboardShortcuts } from '@/hooks/use-keyboard-shortcuts'
import { KeyboardShortcutsDialog } from '@/components/keyboard-shortcuts-dialog'

const navItems = [
  { to: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { to: '/devices', label: 'Devices', icon: Monitor },
  { to: '/topology', label: 'Topology', icon: Network },
  { to: '/agents', label: 'Agents', icon: Bot },
  { to: '/services', label: 'Services', icon: Layers },
  { to: '/documentation', label: 'Docs', icon: FileText },
  { to: '/settings', label: 'Settings', icon: Settings },
  { to: '/about', label: 'About', icon: Info },
]

export function AppLayout() {
  const logout = useAuthStore((s) => s.logout)
  const user = useAuthStore((s) => s.user)
  const navigate = useNavigate()
  const [shortcutsOpen, setShortcutsOpen] = useState(false)

  const openShortcuts = useCallback(() => setShortcutsOpen(true), [])
  const goToDashboard = useCallback(() => navigate('/dashboard'), [navigate])
  const goToDevices = useCallback(() => navigate('/devices'), [navigate])
  const goToTopology = useCallback(() => navigate('/topology'), [navigate])
  const goToAgents = useCallback(() => navigate('/agents'), [navigate])
  const goToServices = useCallback(() => navigate('/services'), [navigate])
  const goToDocs = useCallback(() => navigate('/documentation'), [navigate])
  const goToSettings = useCallback(() => navigate('/settings'), [navigate])

  const shortcuts = useMemo(
    () => [
      { key: '?', shift: true, handler: openShortcuts, description: 'Show keyboard shortcuts' },
      { key: '1', handler: goToDashboard, description: 'Go to Dashboard' },
      { key: '2', handler: goToDevices, description: 'Go to Devices' },
      { key: '3', handler: goToTopology, description: 'Go to Topology' },
      { key: '4', handler: goToAgents, description: 'Go to Agents' },
      { key: '5', handler: goToServices, description: 'Go to Services' },
      { key: '6', handler: goToDocs, description: 'Go to Docs' },
      { key: '7', handler: goToSettings, description: 'Go to Settings' },
    ],
    [openShortcuts, goToDashboard, goToDevices, goToTopology, goToAgents, goToServices, goToDocs, goToSettings]
  )

  useKeyboardShortcuts(shortcuts)

  return (
    <div className="flex min-h-screen">
      <aside className="hidden w-64 flex-col border-r border-[var(--nv-border-subtle)] bg-[var(--nv-sidebar-bg)] md:flex">
        <div className="flex items-center gap-3 border-b border-[var(--nv-border-subtle)] p-4">
          <img src="/favicon.svg" alt="SubNetree" className="h-8 w-8" />
          <h1 className="text-lg font-semibold text-[var(--nv-green-400)]">SubNetree</h1>
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
      <KeyboardShortcutsDialog open={shortcutsOpen} onOpenChange={setShortcutsOpen} />
    </div>
  )
}
