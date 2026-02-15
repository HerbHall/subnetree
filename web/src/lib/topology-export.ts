import { toSvg } from 'html-to-image'

export interface ExportOptions {
  filename?: string
  backgroundColor?: string
}

/**
 * Exports the React Flow viewport as an SVG file.
 * Uses html-to-image's toSvg which serializes the DOM to SVG.
 */
export async function exportTopologyAsSvg(
  element: HTMLElement,
  options: ExportOptions = {},
): Promise<void> {
  const { filename = 'topology.svg', backgroundColor } = options

  const svgDataUrl = await toSvg(element, {
    filter: (node: Node) => {
      // Exclude minimap and controls from export
      if (node instanceof HTMLElement) {
        const classes = node.className?.toString() || ''
        if (
          classes.includes('react-flow__minimap') ||
          classes.includes('react-flow__controls')
        ) {
          return false
        }
      }
      return true
    },
    backgroundColor: backgroundColor || 'transparent',
  })

  // Trigger download via temporary anchor element
  const link = document.createElement('a')
  link.download = filename
  link.href = svgDataUrl
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}
