import { createBrowserRouter, Navigate } from 'react-router-dom'
import { ProtectedRoute } from './protected-route'
import { AppLayout } from '@/layouts/app-layout'
import { AuthLayout } from '@/layouts/auth-layout'
import { LoginPage } from '@/pages/login'
import { SetupPage } from '@/pages/setup'
import { DashboardPage } from '@/pages/dashboard'
import { DevicesPage } from '@/pages/devices'
import { DeviceDetailPage } from '@/pages/devices/[id]'
import { SettingsPage } from '@/pages/settings'
import { AboutPage } from '@/pages/about'
import { NotFoundPage } from '@/pages/not-found'

export const router = createBrowserRouter([
  {
    element: <AuthLayout />,
    children: [
      { path: '/login', element: <LoginPage /> },
      { path: '/setup', element: <SetupPage /> },
    ],
  },
  {
    element: <ProtectedRoute />,
    children: [
      {
        element: <AppLayout />,
        children: [
          { path: '/', element: <Navigate to="/dashboard" replace /> },
          { path: '/dashboard', element: <DashboardPage /> },
          { path: '/devices', element: <DevicesPage /> },
          { path: '/devices/:id', element: <DeviceDetailPage /> },
          { path: '/settings', element: <SettingsPage /> },
          { path: '/about', element: <AboutPage /> },
        ],
      },
    ],
  },
  { path: '*', element: <NotFoundPage /> },
])
