import { describe, it, expect, vi, beforeEach } from 'vitest'
import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { DeviceDetailPage } from './[id]'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render } from '@testing-library/react'
import type { Device, Scan } from '@/api/types'
import type { DeviceStatusEvent } from '@/api/devices'

// Mock device API
vi.mock('@/api/devices', () => ({
  getDevice: vi.fn(),
  updateDevice: vi.fn(),
  getDeviceStatusHistory: vi.fn(),
  getDeviceScanHistory: vi.fn(),
}))

const mockDevice: Device = {
  id: 'device-123',
  hostname: 'test-server',
  ip_addresses: ['192.168.1.100', '192.168.1.101'],
  mac_address: 'AA:BB:CC:DD:EE:FF',
  manufacturer: 'Test Manufacturer',
  device_type: 'server',
  os: 'Ubuntu 22.04',
  status: 'online',
  discovery_method: 'icmp',
  first_seen: '2024-01-15T10:30:00Z',
  last_seen: '2024-01-20T15:45:00Z',
  notes: 'Production web server',
  tags: ['production', 'web-server'],
}

const mockStatusHistory: DeviceStatusEvent[] = [
  { id: 'event-1', device_id: 'device-123', status: 'online', timestamp: '2024-01-20T15:45:00Z' },
  { id: 'event-2', device_id: 'device-123', status: 'offline', timestamp: '2024-01-20T14:30:00Z' },
  { id: 'event-3', device_id: 'device-123', status: 'online', timestamp: '2024-01-20T10:00:00Z' },
]

const mockScanHistory: Scan[] = [
  {
    id: 'scan-1',
    status: 'completed',
    target_cidr: '192.168.1.0/24',
    started_at: '2024-01-20T15:00:00Z',
    completed_at: '2024-01-20T15:05:00Z',
    devices_found: 10,
  },
  {
    id: 'scan-2',
    status: 'completed',
    target_cidr: '192.168.1.0/24',
    started_at: '2024-01-19T12:00:00Z',
    completed_at: '2024-01-19T12:03:00Z',
    devices_found: 8,
  },
]

function renderWithProviders(deviceId = 'device-123') {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  })

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[`/devices/${deviceId}`]}>
        <Routes>
          <Route path="/devices/:id" element={<DeviceDetailPage />} />
          <Route path="/devices" element={<div>Device List</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  )
}

describe('DeviceDetailPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('shows loading skeleton while fetching device', async () => {
    const { getDevice, getDeviceStatusHistory, getDeviceScanHistory } = await import(
      '@/api/devices'
    )
    vi.mocked(getDevice).mockImplementation(() => new Promise(() => {})) // Never resolves
    vi.mocked(getDeviceStatusHistory).mockResolvedValue([])
    vi.mocked(getDeviceScanHistory).mockResolvedValue([])

    renderWithProviders()

    // Should show skeleton (animated pulse elements)
    expect(document.querySelector('.animate-pulse')).toBeInTheDocument()
  })

  it('displays device information correctly', async () => {
    const { getDevice, getDeviceStatusHistory, getDeviceScanHistory } = await import(
      '@/api/devices'
    )
    vi.mocked(getDevice).mockResolvedValue(mockDevice)
    vi.mocked(getDeviceStatusHistory).mockResolvedValue(mockStatusHistory)
    vi.mocked(getDeviceScanHistory).mockResolvedValue(mockScanHistory)

    renderWithProviders()

    await waitFor(() => {
      expect(screen.getByText('test-server')).toBeInTheDocument()
    })

    // Check header information - use getAllByText since "Server" appears multiple times
    const serverLabels = screen.getAllByText('Server')
    expect(serverLabels.length).toBeGreaterThan(0)

    // Check network information
    expect(screen.getByText('192.168.1.100')).toBeInTheDocument()
    expect(screen.getByText('192.168.1.101')).toBeInTheDocument()
    expect(screen.getByText('AA:BB:CC:DD:EE:FF')).toBeInTheDocument()

    // Check device info
    expect(screen.getByText('Test Manufacturer')).toBeInTheDocument()
    expect(screen.getByText('Ubuntu 22.04')).toBeInTheDocument()
  })

  it('displays first_seen and last_seen timestamps', async () => {
    const { getDevice, getDeviceStatusHistory, getDeviceScanHistory } = await import(
      '@/api/devices'
    )
    vi.mocked(getDevice).mockResolvedValue(mockDevice)
    vi.mocked(getDeviceStatusHistory).mockResolvedValue([])
    vi.mocked(getDeviceScanHistory).mockResolvedValue([])

    renderWithProviders()

    await waitFor(() => {
      expect(screen.getByText('test-server')).toBeInTheDocument()
    })

    // Check that first seen and last seen sections exist
    expect(screen.getByText('First Seen')).toBeInTheDocument()
    expect(screen.getByText('Last Seen')).toBeInTheDocument()
  })

  it('displays notes and tags', async () => {
    const { getDevice, getDeviceStatusHistory, getDeviceScanHistory } = await import(
      '@/api/devices'
    )
    vi.mocked(getDevice).mockResolvedValue(mockDevice)
    vi.mocked(getDeviceStatusHistory).mockResolvedValue([])
    vi.mocked(getDeviceScanHistory).mockResolvedValue([])

    renderWithProviders()

    await waitFor(() => {
      expect(screen.getByText('test-server')).toBeInTheDocument()
    })

    // Check notes
    expect(screen.getByText('Production web server')).toBeInTheDocument()

    // Check tags
    expect(screen.getByText('production')).toBeInTheDocument()
    expect(screen.getByText('web-server')).toBeInTheDocument()
  })

  it('displays status timeline', async () => {
    const { getDevice, getDeviceStatusHistory, getDeviceScanHistory } = await import(
      '@/api/devices'
    )
    vi.mocked(getDevice).mockResolvedValue(mockDevice)
    vi.mocked(getDeviceStatusHistory).mockResolvedValue(mockStatusHistory)
    vi.mocked(getDeviceScanHistory).mockResolvedValue([])

    renderWithProviders()

    await waitFor(() => {
      expect(screen.getByText('test-server')).toBeInTheDocument()
    })

    // Check status timeline section exists
    expect(screen.getByText('Status Timeline')).toBeInTheDocument()

    // Check status history entries (there should be multiple "Online" and one "Offline")
    const statusCards = screen.getAllByText(/Online|Offline/)
    expect(statusCards.length).toBeGreaterThan(1)
  })

  it('displays scan history', async () => {
    const { getDevice, getDeviceStatusHistory, getDeviceScanHistory } = await import(
      '@/api/devices'
    )
    vi.mocked(getDevice).mockResolvedValue(mockDevice)
    vi.mocked(getDeviceStatusHistory).mockResolvedValue([])
    vi.mocked(getDeviceScanHistory).mockResolvedValue(mockScanHistory)

    renderWithProviders()

    await waitFor(() => {
      expect(screen.getByText('test-server')).toBeInTheDocument()
    })

    // Check scan history section exists
    expect(screen.getByText('Scan History')).toBeInTheDocument()

    // Check scan history entries
    const cidrElements = screen.getAllByText('192.168.1.0/24')
    expect(cidrElements.length).toBe(2)
  })

  it('shows empty state for status timeline when no history', async () => {
    const { getDevice, getDeviceStatusHistory, getDeviceScanHistory } = await import(
      '@/api/devices'
    )
    vi.mocked(getDevice).mockResolvedValue(mockDevice)
    vi.mocked(getDeviceStatusHistory).mockResolvedValue([])
    vi.mocked(getDeviceScanHistory).mockResolvedValue([])

    renderWithProviders()

    await waitFor(() => {
      expect(screen.getByText('test-server')).toBeInTheDocument()
    })

    expect(screen.getByText('No status history available yet.')).toBeInTheDocument()
  })

  it('shows empty state for scan history when no scans', async () => {
    const { getDevice, getDeviceStatusHistory, getDeviceScanHistory } = await import(
      '@/api/devices'
    )
    vi.mocked(getDevice).mockResolvedValue(mockDevice)
    vi.mocked(getDeviceStatusHistory).mockResolvedValue([])
    vi.mocked(getDeviceScanHistory).mockResolvedValue([])

    renderWithProviders()

    await waitFor(() => {
      expect(screen.getByText('test-server')).toBeInTheDocument()
    })

    expect(screen.getByText('No scan history available yet.')).toBeInTheDocument()
  })

  it('shows error state when device not found', async () => {
    const { getDevice, getDeviceStatusHistory, getDeviceScanHistory } = await import(
      '@/api/devices'
    )
    vi.mocked(getDevice).mockRejectedValue(new Error('Device not found'))
    vi.mocked(getDeviceStatusHistory).mockResolvedValue([])
    vi.mocked(getDeviceScanHistory).mockResolvedValue([])

    renderWithProviders()

    await waitFor(() => {
      expect(screen.getByText('Device not found')).toBeInTheDocument()
    })

    expect(screen.getByText('Return to Device List')).toBeInTheDocument()
  })

  it('has back navigation to device list', async () => {
    const { getDevice, getDeviceStatusHistory, getDeviceScanHistory } = await import(
      '@/api/devices'
    )
    vi.mocked(getDevice).mockResolvedValue(mockDevice)
    vi.mocked(getDeviceStatusHistory).mockResolvedValue([])
    vi.mocked(getDeviceScanHistory).mockResolvedValue([])

    renderWithProviders()

    await waitFor(() => {
      expect(screen.getByText('test-server')).toBeInTheDocument()
    })

    const backLink = screen.getByText('Back to Devices')
    expect(backLink).toBeInTheDocument()
    expect(backLink.closest('a')).toHaveAttribute('href', '/devices')
  })

  it('displays hostname in header when available', async () => {
    const { getDevice, getDeviceStatusHistory, getDeviceScanHistory } = await import(
      '@/api/devices'
    )
    vi.mocked(getDevice).mockResolvedValue(mockDevice)
    vi.mocked(getDeviceStatusHistory).mockResolvedValue([])
    vi.mocked(getDeviceScanHistory).mockResolvedValue([])

    renderWithProviders()

    await waitFor(() => {
      expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('test-server')
    })
  })

  it('displays IP address in header when no hostname', async () => {
    const { getDevice, getDeviceStatusHistory, getDeviceScanHistory } = await import(
      '@/api/devices'
    )
    const deviceWithoutHostname = { ...mockDevice, hostname: '' }
    vi.mocked(getDevice).mockResolvedValue(deviceWithoutHostname)
    vi.mocked(getDeviceStatusHistory).mockResolvedValue([])
    vi.mocked(getDeviceScanHistory).mockResolvedValue([])

    renderWithProviders()

    await waitFor(() => {
      expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('192.168.1.100')
    })
  })

  it('allows entering edit mode for notes', async () => {
    const user = userEvent.setup()
    const { getDevice, getDeviceStatusHistory, getDeviceScanHistory, updateDevice } = await import(
      '@/api/devices'
    )
    vi.mocked(getDevice).mockResolvedValue(mockDevice)
    vi.mocked(getDeviceStatusHistory).mockResolvedValue([])
    vi.mocked(getDeviceScanHistory).mockResolvedValue([])
    vi.mocked(updateDevice).mockResolvedValue(mockDevice)

    renderWithProviders()

    await waitFor(() => {
      expect(screen.getByText('test-server')).toBeInTheDocument()
    })

    // Find the Edit button in the Notes section
    const notesCard = screen.getByText('Notes').closest('.rounded-lg')
    expect(notesCard).toBeInTheDocument()

    const editButtons = within(notesCard as HTMLElement).getAllByRole('button', { name: /edit/i })
    await user.click(editButtons[0])

    // Should show textarea with current notes
    expect(screen.getByPlaceholderText('Add notes about this device...')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /save/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /cancel/i })).toBeInTheDocument()
  })

  it('allows entering edit mode for tags', async () => {
    const user = userEvent.setup()
    const { getDevice, getDeviceStatusHistory, getDeviceScanHistory, updateDevice } = await import(
      '@/api/devices'
    )
    vi.mocked(getDevice).mockResolvedValue(mockDevice)
    vi.mocked(getDeviceStatusHistory).mockResolvedValue([])
    vi.mocked(getDeviceScanHistory).mockResolvedValue([])
    vi.mocked(updateDevice).mockResolvedValue(mockDevice)

    renderWithProviders()

    await waitFor(() => {
      expect(screen.getByText('test-server')).toBeInTheDocument()
    })

    // Find the Tags section (it's after Notes)
    const tagsCard = screen.getByText('Tags').closest('.rounded-lg')
    expect(tagsCard).toBeInTheDocument()

    const editButtons = within(tagsCard as HTMLElement).getAllByRole('button', { name: /edit/i })
    await user.click(editButtons[0])

    // Should show input for tags
    expect(
      screen.getByPlaceholderText(
        'Enter tags separated by commas (e.g., production, critical, web-server)'
      )
    ).toBeInTheDocument()
  })

  it('shows quick access buttons for device with IP', async () => {
    const { getDevice, getDeviceStatusHistory, getDeviceScanHistory } = await import(
      '@/api/devices'
    )
    vi.mocked(getDevice).mockResolvedValue(mockDevice)
    vi.mocked(getDeviceStatusHistory).mockResolvedValue([])
    vi.mocked(getDeviceScanHistory).mockResolvedValue([])

    renderWithProviders()

    await waitFor(() => {
      expect(screen.getByText('test-server')).toBeInTheDocument()
    })

    // Check Quick Access section
    expect(screen.getByText('Quick Access')).toBeInTheDocument()
    expect(screen.getByText('HTTP (192.168.1.100)')).toBeInTheDocument()
    expect(screen.getByText('HTTPS (192.168.1.100)')).toBeInTheDocument()
  })

  it('shows device type icon in header', async () => {
    const { getDevice, getDeviceStatusHistory, getDeviceScanHistory } = await import(
      '@/api/devices'
    )
    vi.mocked(getDevice).mockResolvedValue(mockDevice)
    vi.mocked(getDeviceStatusHistory).mockResolvedValue([])
    vi.mocked(getDeviceScanHistory).mockResolvedValue([])

    renderWithProviders()

    await waitFor(() => {
      expect(screen.getByText('test-server')).toBeInTheDocument()
    })

    // Check for Server type label (appears multiple times on page)
    const serverLabels = screen.getAllByText('Server')
    expect(serverLabels.length).toBeGreaterThan(0)
  })
})
