import { useState, useEffect, useRef } from 'react'
import { createPortal } from 'react-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { X, Plus, Loader2, ChevronDown, ChevronRight } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { createDevice } from '@/api/devices'
import type { DeviceType } from '@/api/types'

interface CreateDeviceDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

const DEVICE_TYPE_OPTIONS: { value: DeviceType; label: string }[] = [
  { value: 'server', label: 'Server' },
  { value: 'desktop', label: 'Desktop' },
  { value: 'laptop', label: 'Laptop' },
  { value: 'mobile', label: 'Mobile' },
  { value: 'router', label: 'Router' },
  { value: 'switch', label: 'Switch' },
  { value: 'access_point', label: 'Access Point' },
  { value: 'firewall', label: 'Firewall' },
  { value: 'printer', label: 'Printer' },
  { value: 'nas', label: 'NAS' },
  { value: 'iot', label: 'IoT' },
  { value: 'phone', label: 'Phone' },
  { value: 'tablet', label: 'Tablet' },
  { value: 'camera', label: 'Camera' },
  { value: 'unknown', label: 'Unknown' },
]

export function CreateDeviceDialog({ open, onOpenChange }: CreateDeviceDialogProps) {
  const queryClient = useQueryClient()
  const dialogRef = useRef<HTMLDivElement>(null)

  const [hostname, setHostname] = useState('')
  const [ipAddresses, setIpAddresses] = useState('')
  const [macAddress, setMacAddress] = useState('')
  const [deviceType, setDeviceType] = useState<DeviceType>('unknown')
  const [notes, setNotes] = useState('')
  const [tags, setTags] = useState('')
  const [validationError, setValidationError] = useState('')
  const [showInventory, setShowInventory] = useState(false)
  const [location, setLocation] = useState('')
  const [category, setCategory] = useState('')
  const [primaryRole, setPrimaryRole] = useState('')
  const [owner, setOwner] = useState('')

  const createMutation = useMutation({
    mutationFn: createDevice,
    onSuccess: (device) => {
      queryClient.invalidateQueries({ queryKey: ['devices'] })
      queryClient.invalidateQueries({ queryKey: ['topology'] })
      queryClient.invalidateQueries({ queryKey: ['inventorySummary'] })
      toast.success(`Device "${device.hostname}" created`)
      resetForm()
      onOpenChange(false)
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : 'Failed to create device')
    },
  })

  function resetForm() {
    setHostname('')
    setIpAddresses('')
    setMacAddress('')
    setDeviceType('unknown')
    setNotes('')
    setTags('')
    setValidationError('')
    setShowInventory(false)
    setLocation('')
    setCategory('')
    setPrimaryRole('')
    setOwner('')
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setValidationError('')

    if (!hostname.trim()) {
      setValidationError('Hostname is required')
      return
    }

    const ips = ipAddresses
      .split(',')
      .map((ip) => ip.trim())
      .filter((ip) => ip.length > 0)

    const tagList = tags
      .split(',')
      .map((t) => t.trim())
      .filter((t) => t.length > 0)

    createMutation.mutate({
      hostname: hostname.trim(),
      ip_addresses: ips,
      mac_address: macAddress.trim() || undefined,
      device_type: deviceType,
      notes: notes.trim() || undefined,
      tags: tagList.length > 0 ? tagList : undefined,
      location: location.trim() || undefined,
      category: category || undefined,
      primary_role: primaryRole.trim() || undefined,
      owner: owner.trim() || undefined,
    })
  }

  // Close on Escape
  useEffect(() => {
    if (!open) return

    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        e.preventDefault()
        onOpenChange(false)
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [open, onOpenChange])

  // Prevent body scroll and focus dialog when open
  useEffect(() => {
    if (!open) return

    const prevOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'

    requestAnimationFrame(() => {
      dialogRef.current?.focus()
    })

    return () => {
      document.body.style.overflow = prevOverflow
    }
  }, [open])

  // Reset form when dialog closes
  useEffect(() => {
    if (!open) {
      resetForm()
    }
  }, [open])

  if (!open) return null

  return createPortal(
    <div
      className="fixed inset-0 z-50 flex items-center justify-center"
      role="dialog"
      aria-modal="true"
      aria-label="Create device"
    >
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/60 backdrop-blur-sm"
        onClick={() => onOpenChange(false)}
      />

      {/* Dialog content */}
      <div
        ref={dialogRef}
        tabIndex={-1}
        className="relative z-50 w-full max-w-lg rounded-lg border bg-card p-6 shadow-lg outline-none max-h-[90vh] overflow-y-auto"
      >
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div>
            <h2 className="text-lg font-semibold">Add Device</h2>
            <p className="text-sm text-muted-foreground mt-0.5">
              Manually register a device on your network.
            </p>
          </div>
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            onClick={() => onOpenChange(false)}
          >
            <X className="h-4 w-4" />
          </Button>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Hostname */}
          <div className="space-y-2">
            <Label htmlFor="hostname">
              Hostname <span className="text-red-500">*</span>
            </Label>
            <Input
              id="hostname"
              placeholder="e.g., nas-01, pi-hole, proxmox-node"
              value={hostname}
              onChange={(e) => setHostname(e.target.value)}
              autoFocus
            />
          </div>

          {/* IP Addresses */}
          <div className="space-y-2">
            <Label htmlFor="ip-addresses">IP Addresses</Label>
            <Input
              id="ip-addresses"
              placeholder="e.g., 192.168.1.100, 10.0.0.5"
              value={ipAddresses}
              onChange={(e) => setIpAddresses(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              Separate multiple addresses with commas.
            </p>
          </div>

          {/* MAC Address */}
          <div className="space-y-2">
            <Label htmlFor="mac-address">MAC Address</Label>
            <Input
              id="mac-address"
              placeholder="e.g., AA:BB:CC:DD:EE:FF"
              value={macAddress}
              onChange={(e) => setMacAddress(e.target.value)}
            />
          </div>

          {/* Device Type */}
          <div className="space-y-2">
            <Label htmlFor="device-type">Device Type</Label>
            <select
              id="device-type"
              value={deviceType}
              onChange={(e) => setDeviceType(e.target.value as DeviceType)}
              className="w-full h-9 px-3 rounded-md border bg-background text-sm"
            >
              {DEVICE_TYPE_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </div>

          {/* Notes */}
          <div className="space-y-2">
            <Label htmlFor="notes">Notes</Label>
            <textarea
              id="notes"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              className="w-full min-h-[80px] p-3 rounded-md border bg-background text-sm resize-y"
              placeholder="Optional notes about this device..."
            />
          </div>

          {/* Tags */}
          <div className="space-y-2">
            <Label htmlFor="tags">Tags</Label>
            <Input
              id="tags"
              placeholder="e.g., production, critical, lab"
              value={tags}
              onChange={(e) => setTags(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              Separate tags with commas.
            </p>
          </div>

          {/* Inventory Details (collapsible) */}
          <div className="border rounded-md">
            <button
              type="button"
              onClick={() => setShowInventory(!showInventory)}
              className="flex items-center justify-between w-full px-3 py-2 text-sm font-medium text-muted-foreground hover:text-foreground transition-colors"
            >
              <span>Inventory Details</span>
              {showInventory ? (
                <ChevronDown className="h-4 w-4" />
              ) : (
                <ChevronRight className="h-4 w-4" />
              )}
            </button>
            {showInventory && (
              <div className="px-3 pb-3 space-y-4 border-t pt-3">
                {/* Location */}
                <div className="space-y-2">
                  <Label htmlFor="location">Location</Label>
                  <Input
                    id="location"
                    placeholder="e.g., Server Room A, Office 2F"
                    value={location}
                    onChange={(e) => setLocation(e.target.value)}
                  />
                </div>

                {/* Category */}
                <div className="space-y-2">
                  <Label htmlFor="category">Category</Label>
                  <select
                    id="category"
                    value={category}
                    onChange={(e) => setCategory(e.target.value)}
                    className="w-full h-9 px-3 rounded-md border bg-background text-sm"
                  >
                    <option value="">None</option>
                    <option value="production">Production</option>
                    <option value="development">Development</option>
                    <option value="network">Network</option>
                    <option value="storage">Storage</option>
                    <option value="iot">IoT</option>
                    <option value="personal">Personal</option>
                    <option value="shared">Shared</option>
                    <option value="other">Other</option>
                  </select>
                </div>

                {/* Primary Role */}
                <div className="space-y-2">
                  <Label htmlFor="primary-role">Primary Role</Label>
                  <Input
                    id="primary-role"
                    placeholder="e.g., Web Server, DNS, File Share"
                    value={primaryRole}
                    onChange={(e) => setPrimaryRole(e.target.value)}
                  />
                </div>

                {/* Owner */}
                <div className="space-y-2">
                  <Label htmlFor="owner">Owner</Label>
                  <Input
                    id="owner"
                    placeholder="e.g., John Doe, IT Team"
                    value={owner}
                    onChange={(e) => setOwner(e.target.value)}
                  />
                </div>
              </div>
            )}
          </div>

          {/* Validation error */}
          {validationError && (
            <p className="text-sm text-red-500">{validationError}</p>
          )}

          {/* Actions */}
          <div className="flex items-center justify-end gap-2 pt-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={createMutation.isPending}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={createMutation.isPending}
              className="gap-2"
            >
              {createMutation.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Plus className="h-4 w-4" />
              )}
              {createMutation.isPending ? 'Creating...' : 'Create Device'}
            </Button>
          </div>
        </form>
      </div>
    </div>,
    document.body
  )
}
