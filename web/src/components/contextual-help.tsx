import { Info } from 'lucide-react'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'

/**
 * Small info icon with tooltip. Use next to labels and column headers.
 */
export function HelpIcon({ content }: { content: string }) {
  return (
    <TooltipProvider delayDuration={300}>
      <Tooltip>
        <TooltipTrigger asChild>
          <Info className="h-3.5 w-3.5 text-muted-foreground/60 hover:text-muted-foreground cursor-help inline-block ml-1" />
        </TooltipTrigger>
        <TooltipContent className="max-w-xs text-xs">
          <p>{content}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}

/**
 * Info icon that opens a popover with richer content. Use for complex concepts.
 */
export function HelpPopover({
  title,
  children,
}: {
  title: string
  children: React.ReactNode
}) {
  return (
    <Popover>
      <PopoverTrigger asChild>
        <Info className="h-3.5 w-3.5 text-muted-foreground/60 hover:text-muted-foreground cursor-help inline-block ml-1" />
      </PopoverTrigger>
      <PopoverContent className="text-sm space-y-2">
        <p className="font-medium">{title}</p>
        {children}
      </PopoverContent>
    </Popover>
  )
}

/**
 * Small helper text below form fields.
 */
export function FieldHelp({ text }: { text: string }) {
  return (
    <p className="text-xs text-muted-foreground/70 mt-1">{text}</p>
  )
}
