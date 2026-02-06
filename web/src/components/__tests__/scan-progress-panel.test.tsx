import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ScanProgressPanel } from '../scan-progress-panel'
import type { ScanProgress } from '@/stores/scan'
import type { Device } from '@/api/types'

/** Create a minimal Device with only required fields for testing. */
function testDevice(overrides: Partial<Device> & { id: string }): Device {
  return {
    ip_addresses: [],
    hostname: '',
    mac_address: '',
    manufacturer: '',
    device_type: 'unknown',
    os: '',
    status: 'online',
    discovery_method: 'scan',
    first_seen: new Date().toISOString(),
    last_seen: new Date().toISOString(),
    ...overrides,
  }
}

describe('ScanProgressPanel', () => {
  const createBaseScan = (overrides?: Partial<ScanProgress>): ScanProgress => ({
    scanId: 'test-scan-1',
    targetCidr: '192.168.1.0/24',
    status: 'scanning',
    devicesFound: 0,
    subnetSize: 256,
    hostsAlive: 0,
    newDevices: [],
    startedAt: new Date(),
    ...overrides,
  })

  it('renders nothing when activeScan is null', () => {
    const { container } = render(<ScanProgressPanel activeScan={null} progress={0} />)
    expect(container).toBeEmptyDOMElement()
  })

  it('shows scanning state with target CIDR during starting phase', () => {
    const scan = createBaseScan({ status: 'starting' })
    render(<ScanProgressPanel activeScan={scan} progress={0} />)

    expect(screen.getByText('Scanning 192.168.1.0/24...')).toBeInTheDocument()
  })

  it('shows scanning state with target CIDR during scanning phase', () => {
    const scan = createBaseScan({ status: 'scanning' })
    render(<ScanProgressPanel activeScan={scan} progress={0} />)

    expect(screen.getByText('Scanning 192.168.1.0/24...')).toBeInTheDocument()
  })

  it('shows scanning state with target CIDR during processing phase', () => {
    const scan = createBaseScan({ status: 'processing', hostsAlive: 10 })
    render(<ScanProgressPanel activeScan={scan} progress={50} />)

    expect(screen.getByText('Scanning 192.168.1.0/24...')).toBeInTheDocument()
  })

  it('shows "ping sweep in progress" during scanning phase', () => {
    const scan = createBaseScan({ status: 'scanning', devicesFound: 3 })
    render(<ScanProgressPanel activeScan={scan} progress={0} />)

    expect(screen.getByText('3 devices found (ping sweep in progress)')).toBeInTheDocument()
  })

  it('shows progress percentage during processing phase', () => {
    const scan = createBaseScan({
      status: 'processing',
      devicesFound: 5,
      hostsAlive: 10,
    })
    render(<ScanProgressPanel activeScan={scan} progress={50} />)

    expect(screen.getByText('50%')).toBeInTheDocument()
    expect(screen.getByText('5 devices found of 10 alive hosts')).toBeInTheDocument()
  })

  it('shows device count during starting phase', () => {
    const scan = createBaseScan({ status: 'starting', devicesFound: 0 })
    render(<ScanProgressPanel activeScan={scan} progress={0} />)

    expect(screen.getByText('0 devices found')).toBeInTheDocument()
  })

  it('shows singular "device" when count is 1', () => {
    const scan = createBaseScan({ status: 'scanning', devicesFound: 1 })
    render(<ScanProgressPanel activeScan={scan} progress={0} />)

    expect(screen.getByText('1 device found (ping sweep in progress)')).toBeInTheDocument()
  })

  it('shows plural "devices" when count is not 1', () => {
    const scan = createBaseScan({ status: 'scanning', devicesFound: 2 })
    render(<ScanProgressPanel activeScan={scan} progress={0} />)

    expect(screen.getByText('2 devices found (ping sweep in progress)')).toBeInTheDocument()
  })

  it('shows completion summary with device count and target', () => {
    const scan = createBaseScan({ status: 'completed', devicesFound: 5 })
    render(<ScanProgressPanel activeScan={scan} progress={100} />)

    expect(screen.getByText('Scan Completed')).toBeInTheDocument()
    expect(screen.getByText('Found 5 devices in 192.168.1.0/24')).toBeInTheDocument()
  })

  it('shows singular "device" in completion summary when count is 1', () => {
    const scan = createBaseScan({ status: 'completed', devicesFound: 1 })
    render(<ScanProgressPanel activeScan={scan} progress={100} />)

    expect(screen.getByText('Found 1 device in 192.168.1.0/24')).toBeInTheDocument()
  })

  it('shows error message when status is error', () => {
    const scan = createBaseScan({
      status: 'error',
      error: 'Network unreachable',
    })
    render(<ScanProgressPanel activeScan={scan} progress={0} />)

    expect(screen.getByText('Scan Failed')).toBeInTheDocument()
    expect(screen.getByText('Network unreachable')).toBeInTheDocument()
  })

  it('shows default error message when error string is empty', () => {
    const scan = createBaseScan({
      status: 'error',
      error: undefined,
    })
    render(<ScanProgressPanel activeScan={scan} progress={0} />)

    expect(screen.getByText('Scan Failed')).toBeInTheDocument()
    expect(screen.getByText('An error occurred during the scan')).toBeInTheDocument()
  })

  it('shows recently found devices with IP addresses', () => {
    const scan = createBaseScan({
      status: 'scanning',
      devicesFound: 3,
      newDevices: [
        testDevice({ id: 'device-1', ip_addresses: ['192.168.1.10'], hostname: 'server-1', manufacturer: 'Dell' }),
        testDevice({ id: 'device-2', ip_addresses: ['192.168.1.20'], hostname: 'laptop-2' }),
        testDevice({ id: 'device-3', ip_addresses: ['192.168.1.30'] }),
      ],
    })
    render(<ScanProgressPanel activeScan={scan} progress={0} />)

    expect(screen.getByText('Recently found:')).toBeInTheDocument()
    expect(screen.getByText('192.168.1.10')).toBeInTheDocument()
    expect(screen.getByText('192.168.1.20')).toBeInTheDocument()
    expect(screen.getByText('192.168.1.30')).toBeInTheDocument()
  })

  it('shows hostname when available in recently found devices', () => {
    const scan = createBaseScan({
      status: 'processing',
      devicesFound: 1,
      hostsAlive: 5,
      newDevices: [
        testDevice({ id: 'device-1', ip_addresses: ['192.168.1.10'], hostname: 'web-server' }),
      ],
    })
    render(<ScanProgressPanel activeScan={scan} progress={20} />)

    expect(screen.getByText('web-server')).toBeInTheDocument()
  })

  it('shows manufacturer when available in recently found devices', () => {
    const scan = createBaseScan({
      status: 'scanning',
      devicesFound: 1,
      newDevices: [
        testDevice({ id: 'device-1', ip_addresses: ['192.168.1.10'], manufacturer: 'Apple' }),
      ],
    })
    render(<ScanProgressPanel activeScan={scan} progress={0} />)

    expect(screen.getByText('Apple')).toBeInTheDocument()
  })

  it('does not show recently found section when list is empty', () => {
    const scan = createBaseScan({
      status: 'scanning',
      devicesFound: 0,
      newDevices: [],
    })
    render(<ScanProgressPanel activeScan={scan} progress={0} />)

    expect(screen.queryByText('Recently found:')).not.toBeInTheDocument()
  })

  it('does not show recently found section when scan is completed', () => {
    const scan = createBaseScan({
      status: 'completed',
      devicesFound: 3,
      newDevices: [
        testDevice({ id: 'device-1', ip_addresses: ['192.168.1.10'] }),
      ],
    })
    render(<ScanProgressPanel activeScan={scan} progress={100} />)

    expect(screen.queryByText('Recently found:')).not.toBeInTheDocument()
  })

  it('does not show recently found section when scan has error', () => {
    const scan = createBaseScan({
      status: 'error',
      error: 'Network error',
      devicesFound: 1,
      newDevices: [
        testDevice({ id: 'device-1', ip_addresses: ['192.168.1.10'] }),
      ],
    })
    render(<ScanProgressPanel activeScan={scan} progress={0} />)

    expect(screen.queryByText('Recently found:')).not.toBeInTheDocument()
  })

  it('shows only last 5 devices when more than 5 are found', () => {
    const scan = createBaseScan({
      status: 'scanning',
      devicesFound: 10,
      newDevices: [
        testDevice({ id: 'device-1', ip_addresses: ['192.168.1.1'] }),
        testDevice({ id: 'device-2', ip_addresses: ['192.168.1.2'] }),
        testDevice({ id: 'device-3', ip_addresses: ['192.168.1.3'] }),
        testDevice({ id: 'device-4', ip_addresses: ['192.168.1.4'] }),
        testDevice({ id: 'device-5', ip_addresses: ['192.168.1.5'] }),
        testDevice({ id: 'device-6', ip_addresses: ['192.168.1.6'] }),
        testDevice({ id: 'device-7', ip_addresses: ['192.168.1.7'] }),
        testDevice({ id: 'device-8', ip_addresses: ['192.168.1.8'] }),
      ],
    })
    render(<ScanProgressPanel activeScan={scan} progress={0} />)

    // Should show last 5 devices (4-8), not first 3 (1-3)
    expect(screen.queryByText('192.168.1.1')).not.toBeInTheDocument()
    expect(screen.queryByText('192.168.1.2')).not.toBeInTheDocument()
    expect(screen.queryByText('192.168.1.3')).not.toBeInTheDocument()
    expect(screen.getByText('192.168.1.4')).toBeInTheDocument()
    expect(screen.getByText('192.168.1.5')).toBeInTheDocument()
    expect(screen.getByText('192.168.1.6')).toBeInTheDocument()
    expect(screen.getByText('192.168.1.7')).toBeInTheDocument()
    expect(screen.getByText('192.168.1.8')).toBeInTheDocument()
  })

  it('shows devices in reverse order (newest first)', () => {
    const scan = createBaseScan({
      status: 'scanning',
      devicesFound: 3,
      newDevices: [
        testDevice({ id: 'device-1', ip_addresses: ['192.168.1.1'] }),
        testDevice({ id: 'device-2', ip_addresses: ['192.168.1.2'] }),
        testDevice({ id: 'device-3', ip_addresses: ['192.168.1.3'] }),
      ],
    })
    const { container } = render(<ScanProgressPanel activeScan={scan} progress={0} />)

    const deviceElements = container.querySelectorAll('.font-mono')
    expect(deviceElements).toHaveLength(3)
    // Most recent device (device-3) should be first in the rendered list
    expect(deviceElements[0]).toHaveTextContent('192.168.1.3')
    expect(deviceElements[1]).toHaveTextContent('192.168.1.2')
    expect(deviceElements[2]).toHaveTextContent('192.168.1.1')
  })

  it('does not show progress percentage during starting phase', () => {
    const scan = createBaseScan({ status: 'starting', devicesFound: 0 })
    render(<ScanProgressPanel activeScan={scan} progress={0} />)

    // Text "0%" should not appear
    expect(screen.queryByText(/^\d+%$/)).not.toBeInTheDocument()
  })

  it('does not show progress percentage during scanning phase', () => {
    const scan = createBaseScan({ status: 'scanning', devicesFound: 2 })
    render(<ScanProgressPanel activeScan={scan} progress={30} />)

    // Text "30%" should not appear during scanning
    expect(screen.queryByText('30%')).not.toBeInTheDocument()
  })

  it('does not show progress bar during completed status', () => {
    const scan = createBaseScan({ status: 'completed', devicesFound: 5 })
    render(<ScanProgressPanel activeScan={scan} progress={100} />)

    // Progress bar should not be rendered
    expect(screen.queryByRole('progressbar')).not.toBeInTheDocument()
  })

  it('does not show progress bar during error status', () => {
    const scan = createBaseScan({ status: 'error', error: 'Test error' })
    render(<ScanProgressPanel activeScan={scan} progress={0} />)

    // Progress bar should not be rendered
    expect(screen.queryByRole('progressbar')).not.toBeInTheDocument()
  })
})
