import { lazy, Suspense } from 'react'
import { createBrowserRouter, Navigate } from 'react-router-dom'
import { ProtectedRoute } from './protected-route'
import { AppLayout } from '@/layouts/app-layout'
import { AuthLayout } from '@/layouts/auth-layout'

/* eslint-disable react-refresh/only-export-components */
// Lazy-loaded page components for route-level code splitting
const LoginPage = lazy(() =>
  import('@/pages/login').then((m) => ({ default: m.LoginPage }))
)
const SetupPage = lazy(() =>
  import('@/pages/setup').then((m) => ({ default: m.SetupPage }))
)
const DashboardPage = lazy(() =>
  import('@/pages/dashboard').then((m) => ({ default: m.DashboardPage }))
)
const DevicesPage = lazy(() =>
  import('@/pages/devices').then((m) => ({ default: m.DevicesPage }))
)
const DeviceDetailPage = lazy(() =>
  import('@/pages/devices/[id]').then((m) => ({ default: m.DeviceDetailPage }))
)
const TopologyPage = lazy(() =>
  import('@/pages/topology').then((m) => ({ default: m.TopologyPage }))
)
const AgentsPage = lazy(() =>
  import('@/pages/agents').then((m) => ({ default: m.AgentsPage }))
)
const AgentDetailPage = lazy(() =>
  import('@/pages/agents/[id]').then((m) => ({ default: m.AgentDetailPage }))
)
const MonitoringPage = lazy(() =>
  import('@/pages/monitoring').then((m) => ({ default: m.MonitoringPage }))
)
const ServiceMapPage = lazy(() =>
  import('@/pages/services').then((m) => ({ default: m.ServiceMapPage }))
)
const VaultPage = lazy(() =>
  import('@/pages/vault').then((m) => ({ default: m.VaultPage }))
)
const DocumentationPage = lazy(() =>
  import('@/pages/documentation').then((m) => ({ default: m.DocumentationPage }))
)
const SettingsPage = lazy(() =>
  import('@/pages/settings').then((m) => ({ default: m.SettingsPage }))
)
const AboutPage = lazy(() =>
  import('@/pages/about').then((m) => ({ default: m.AboutPage }))
)
const NotFoundPage = lazy(() =>
  import('@/pages/not-found').then((m) => ({ default: m.NotFoundPage }))
)

function PageLoader() {
  return (
    <div className="flex items-center justify-center h-full min-h-[200px]">
      <div className="text-sm text-[var(--nv-text-muted)]">Loading...</div>
    </div>
  )
}

function SuspensePage({ children }: { children: React.ReactNode }) {
  return <Suspense fallback={<PageLoader />}>{children}</Suspense>
}

export const router = createBrowserRouter([
  {
    element: <AuthLayout />,
    children: [
      { path: '/login', element: <SuspensePage><LoginPage /></SuspensePage> },
      { path: '/setup', element: <SuspensePage><SetupPage /></SuspensePage> },
    ],
  },
  {
    element: <ProtectedRoute />,
    children: [
      {
        element: <AppLayout />,
        children: [
          { path: '/', element: <Navigate to="/dashboard" replace /> },
          { path: '/dashboard', element: <SuspensePage><DashboardPage /></SuspensePage> },
          { path: '/devices', element: <SuspensePage><DevicesPage /></SuspensePage> },
          { path: '/devices/:id', element: <SuspensePage><DeviceDetailPage /></SuspensePage> },
          { path: '/topology', element: <SuspensePage><TopologyPage /></SuspensePage> },
          { path: '/agents', element: <SuspensePage><AgentsPage /></SuspensePage> },
          { path: '/agents/:id', element: <SuspensePage><AgentDetailPage /></SuspensePage> },
          { path: '/monitoring', element: <SuspensePage><MonitoringPage /></SuspensePage> },
          { path: '/services', element: <SuspensePage><ServiceMapPage /></SuspensePage> },
          { path: '/vault', element: <SuspensePage><VaultPage /></SuspensePage> },
          { path: '/documentation', element: <SuspensePage><DocumentationPage /></SuspensePage> },
          { path: '/settings', element: <SuspensePage><SettingsPage /></SuspensePage> },
          { path: '/about', element: <SuspensePage><AboutPage /></SuspensePage> },
        ],
      },
    ],
  },
  { path: '*', element: <SuspensePage><NotFoundPage /></SuspensePage> },
])
