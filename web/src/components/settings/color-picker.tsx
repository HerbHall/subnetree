import { useState, useEffect } from 'react'
import { RotateCcw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

/**
 * Convert CSS color values (rgb, rgba, named colors, etc.) to hex for input[type=color].
 * Returns the original value if conversion isn't possible.
 */
function toHex(value: string): string {
  if (value.startsWith('#')) {
    // Normalize shorthand (#abc -> #aabbcc)
    if (value.length === 4) {
      return `#${value[1]}${value[1]}${value[2]}${value[2]}${value[3]}${value[3]}`
    }
    return value.slice(0, 7) // Strip alpha if present (#rrggbbaa -> #rrggbb)
  }
  // For rgba/rgb/other, try to use a canvas to resolve
  try {
    const canvas = document.createElement('canvas')
    canvas.width = canvas.height = 1
    const ctx = canvas.getContext('2d')
    if (!ctx) return '#000000'
    ctx.fillStyle = value
    ctx.fillRect(0, 0, 1, 1)
    const [r, g, b] = ctx.getImageData(0, 0, 1, 1).data
    return `#${r.toString(16).padStart(2, '0')}${g.toString(16).padStart(2, '0')}${b.toString(16).padStart(2, '0')}`
  } catch {
    return '#000000'
  }
}

function isColorValue(value: string): boolean {
  return (
    value.startsWith('#') ||
    value.startsWith('rgb') ||
    value.startsWith('hsl') ||
    value.startsWith('oklch') ||
    // Common named colors
    /^(red|blue|green|white|black|transparent)$/i.test(value)
  )
}

interface ColorPickerProps {
  varName: string
  value: string
  defaultValue: string
  onChange: (value: string) => void
  onReset: () => void
  isOverridden: boolean
}

export function ColorPicker({
  varName,
  value,
  defaultValue,
  onChange,
  onReset,
  isOverridden,
}: ColorPickerProps) {
  const label = varName.replace('--nv-', '')
  const showColorInput = isColorValue(value) || isColorValue(defaultValue)
  const [textValue, setTextValue] = useState(value)

  useEffect(() => {
    setTextValue(value)
  }, [value])

  function handleTextChange(newValue: string) {
    setTextValue(newValue)
    // Apply live if it looks like a valid color or value
    if (newValue.length >= 3) {
      onChange(newValue)
    }
  }

  function handleTextBlur() {
    if (textValue !== value) {
      onChange(textValue)
    }
  }

  return (
    <div className="flex items-center gap-2 py-1">
      {/* Color swatch / input */}
      {showColorInput && (
        <input
          type="color"
          value={toHex(value)}
          onChange={(e) => {
            onChange(e.target.value)
            setTextValue(e.target.value)
          }}
          className="h-8 w-8 shrink-0 cursor-pointer rounded border border-border bg-transparent p-0.5"
          title={label}
        />
      )}

      {/* Label */}
      <span
        className={`text-xs font-mono min-w-0 flex-1 truncate ${
          isOverridden ? 'text-foreground font-medium' : 'text-muted-foreground'
        }`}
        title={varName}
      >
        {label}
      </span>

      {/* Text value */}
      <Input
        value={textValue}
        onChange={(e) => handleTextChange(e.target.value)}
        onBlur={handleTextBlur}
        className="h-7 w-24 text-xs font-mono px-1.5"
      />

      {/* Reset button */}
      {isOverridden && (
        <Button
          variant="ghost"
          size="icon"
          onClick={onReset}
          className="h-7 w-7 shrink-0"
          title={`Reset to default (${defaultValue})`}
        >
          <RotateCcw className="h-3 w-3" />
        </Button>
      )}
    </div>
  )
}

interface TextValueEditorProps {
  varName: string
  value: string
  defaultValue: string
  onChange: (value: string) => void
  onReset: () => void
  isOverridden: boolean
}

export function TextValueEditor({
  varName,
  value,
  defaultValue,
  onChange,
  onReset,
  isOverridden,
}: TextValueEditorProps) {
  const label = varName.replace('--nv-', '')

  return (
    <div className="flex items-center gap-2 py-1">
      <span
        className={`text-xs font-mono min-w-0 flex-shrink-0 ${
          isOverridden ? 'text-foreground font-medium' : 'text-muted-foreground'
        }`}
        title={varName}
      >
        {label}
      </span>

      <Input
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="h-7 text-xs font-mono px-1.5 flex-1"
        placeholder={defaultValue}
      />

      {isOverridden && (
        <Button
          variant="ghost"
          size="icon"
          onClick={onReset}
          className="h-7 w-7 shrink-0"
          title={`Reset to default`}
        >
          <RotateCcw className="h-3 w-3" />
        </Button>
      )}
    </div>
  )
}
