import { memo, useCallback, useRef } from 'react'
import { Image, Grid3X3, CircleDot, X, Upload } from 'lucide-react'
import { toast } from 'sonner'
import type { BackgroundType, TopologyBackgroundSettings } from './background-storage'

const MAX_IMAGE_SIZE_MB = 5
const MAX_IMAGE_SIZE_BYTES = MAX_IMAGE_SIZE_MB * 1024 * 1024
const ACCEPTED_IMAGE_TYPES = '.png,.jpg,.jpeg,.svg'

interface BackgroundSettingsPanelProps {
  settings: TopologyBackgroundSettings
  onChange: (settings: TopologyBackgroundSettings) => void
}

const bgOptions: { value: BackgroundType; label: string; Icon: typeof Grid3X3 }[] = [
  { value: 'none', label: 'None', Icon: X },
  { value: 'grid', label: 'Grid', Icon: Grid3X3 },
  { value: 'dots', label: 'Dots', Icon: CircleDot },
  { value: 'image', label: 'Floor Plan', Icon: Image },
]

/**
 * Background settings panel for the topology toolbar.
 * Rendered inside a popover, allows choosing background type,
 * adjusting opacity, and uploading a floor plan image.
 */
export const BackgroundSettingsPanel = memo(function BackgroundSettingsPanel({
  settings,
  onChange,
}: BackgroundSettingsPanelProps) {
  const fileInputRef = useRef<HTMLInputElement>(null)

  const handleTypeChange = useCallback(
    (type: BackgroundType) => {
      if (type === 'image' && !settings.imageData) {
        // Prompt file upload when selecting image type without existing image
        fileInputRef.current?.click()
        return
      }
      onChange({ ...settings, type })
    },
    [settings, onChange]
  )

  const handleOpacityChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      onChange({ ...settings, opacity: Number(e.target.value) })
    },
    [settings, onChange]
  )

  const handleFileSelect = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0]
      if (!file) return

      if (file.size > MAX_IMAGE_SIZE_BYTES) {
        toast.error(`Image must be under ${MAX_IMAGE_SIZE_MB}MB`)
        return
      }

      const reader = new FileReader()
      reader.onload = () => {
        const dataUrl = reader.result as string
        onChange({ ...settings, type: 'image', imageData: dataUrl })
        toast.success('Floor plan image loaded')
      }
      reader.onerror = () => {
        toast.error('Failed to read image file')
      }
      reader.readAsDataURL(file)

      // Reset input so the same file can be re-selected
      e.target.value = ''
    },
    [settings, onChange]
  )

  const handleReplaceImage = useCallback(() => {
    fileInputRef.current?.click()
  }, [])

  const handleRemoveImage = useCallback(() => {
    onChange({ ...settings, type: 'none', imageData: undefined })
    toast.success('Floor plan image removed')
  }, [settings, onChange])

  return (
    <div className="w-56">
      <p
        className="text-xs font-medium mb-2"
        style={{ color: 'var(--nv-text-primary)' }}
      >
        Background
      </p>

      {/* Type selector */}
      <div className="flex gap-1 mb-3">
        {bgOptions.map(({ value, label, Icon }) => (
          <button
            key={value}
            onClick={() => handleTypeChange(value)}
            title={label}
            className="flex flex-col items-center gap-0.5 rounded-md px-2 py-1.5 text-[10px] font-medium transition-colors flex-1"
            style={{
              backgroundColor:
                settings.type === value
                  ? 'var(--nv-bg-active)'
                  : 'transparent',
              color:
                settings.type === value
                  ? 'var(--nv-text-accent)'
                  : 'var(--nv-text-secondary)',
            }}
          >
            <Icon className="h-3.5 w-3.5" />
            {label}
          </button>
        ))}
      </div>

      {/* Opacity slider */}
      {settings.type !== 'none' && (
        <div className="mb-3">
          <div className="flex items-center justify-between mb-1">
            <span
              className="text-[10px] font-medium"
              style={{ color: 'var(--nv-text-secondary)' }}
            >
              Opacity
            </span>
            <span
              className="text-[10px] tabular-nums"
              style={{ color: 'var(--nv-text-secondary)' }}
            >
              {settings.opacity}%
            </span>
          </div>
          <input
            type="range"
            min={10}
            max={100}
            step={5}
            value={settings.opacity}
            onChange={handleOpacityChange}
            className="w-full h-1.5 rounded-full appearance-none cursor-pointer
              [&::-webkit-slider-thumb]:appearance-none
              [&::-webkit-slider-thumb]:h-3.5
              [&::-webkit-slider-thumb]:w-3.5
              [&::-webkit-slider-thumb]:rounded-full
              [&::-webkit-slider-thumb]:border-2
              [&::-webkit-slider-thumb]:cursor-pointer"
            style={{
              background: `linear-gradient(to right, var(--nv-green-600) 0%, var(--nv-green-600) ${((settings.opacity - 10) / 90) * 100}%, var(--nv-border-default) ${((settings.opacity - 10) / 90) * 100}%, var(--nv-border-default) 100%)`,
              // Thumb styling via CSS custom properties
              // @ts-expect-error -- CSS custom props for slider thumb
              '--thumb-bg': 'var(--nv-green-400)',
              '--thumb-border': 'var(--nv-bg-card)',
            }}
          />
          <style>{`
            input[type="range"]::-webkit-slider-thumb {
              background: var(--nv-green-400);
              border-color: var(--nv-bg-card);
            }
            input[type="range"]::-moz-range-thumb {
              height: 14px;
              width: 14px;
              border-radius: 50%;
              background: var(--nv-green-400);
              border: 2px solid var(--nv-bg-card);
              cursor: pointer;
            }
            input[type="range"]::-moz-range-track {
              height: 6px;
              border-radius: 9999px;
            }
          `}</style>
        </div>
      )}

      {/* Image controls */}
      {settings.type === 'image' && settings.imageData && (
        <div className="flex gap-1.5">
          <button
            onClick={handleReplaceImage}
            className="flex-1 flex items-center justify-center gap-1 rounded-md px-2 py-1 text-[10px] font-medium transition-colors"
            style={{
              backgroundColor: 'var(--nv-bg-hover)',
              color: 'var(--nv-text-secondary)',
              border: '1px solid var(--nv-border-subtle)',
            }}
          >
            <Upload className="h-3 w-3" />
            Replace
          </button>
          <button
            onClick={handleRemoveImage}
            className="flex-1 flex items-center justify-center gap-1 rounded-md px-2 py-1 text-[10px] font-medium transition-colors"
            style={{
              backgroundColor: 'var(--nv-bg-hover)',
              color: 'var(--nv-text-secondary)',
              border: '1px solid var(--nv-border-subtle)',
            }}
          >
            <X className="h-3 w-3" />
            Remove
          </button>
        </div>
      )}

      {/* Hidden file input */}
      <input
        ref={fileInputRef}
        type="file"
        accept={ACCEPTED_IMAGE_TYPES}
        onChange={handleFileSelect}
        className="hidden"
      />
    </div>
  )
})
