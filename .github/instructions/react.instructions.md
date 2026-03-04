---
applyTo: "web/**/*.{ts,tsx}"
---

# React/TypeScript Coding Instructions

## Component Patterns

Define props interfaces above the component, use functional components:

```tsx
interface ItemCardProps {
    id: string
    label: string
    onSelect: (id: string) => void
}

export function ItemCard({ id, label, onSelect }: ItemCardProps) {
    return <button onClick={() => onSelect(id)}>{label}</button>
}
```

## State Management

- **TanStack Query** for all server state (fetching, caching, mutations)
- **Zustand** or `useState` for client-only UI state
- **No `useEffect` for data syncing** -- use nullable local override instead:

```tsx
// GOOD: nullable override, no useEffect sync
const { data: serverValue } = useQuery({ queryKey: ['config'], queryFn: getConfig })
const [localOverride, setLocalOverride] = useState<string | null>(null)
const displayValue = localOverride ?? serverValue ?? ''

// Reset on save success to re-sync from server
onSuccess: () => { setLocalOverride(null); queryClient.invalidateQueries(['config']) }
```

## API Integration

Always use the typed API layer; never call `fetch` directly from components:

```tsx
// mutations invalidate relevant queries on success
const mutation = useMutation({ mutationFn: updateItem, onSuccess: () =>
    queryClient.invalidateQueries({ queryKey: ['items'] })
})
```

## Union Return Types

When a function returns a union type, add a type guard and use it at EVERY call site:

```tsx
// BAD: TS2339 -- access_token not on union
const result = await loginApi(user, pass)
setToken(result.access_token)

// GOOD: narrow first
const result = await loginApi(user, pass)
if (isMFAChallenge(result)) throw new Error('unexpected MFA')
setToken(result.access_token)
```

## TypeScript

- Strict mode -- no `any`; use `unknown` with type guards
- JSX short-circuit: `{expanded && item.details != null && <div/>}` (use `!= null`, not bare `&&` on `unknown`)
- Unused imports: ESLint catches these even when `tsc` does not -- verify every named import is used

## React Compiler Lint

Do not mutate `ref.current` during render -- wrap in `useEffect`:

```tsx
// BAD: ref mutation during render
onMessageRef.current = onMessage

// GOOD: wrap in effect
useEffect(() => { onMessageRef.current = onMessage }, [onMessage])
```

For Popper/Popover anchor elements, use callback ref with `useState` instead of `useRef`:

```tsx
// GOOD: callback ref avoids reading ref.current during render
const [anchorEl, setAnchorEl] = useState<HTMLButtonElement | null>(null)
<Button ref={setAnchorEl}>Menu</Button>
<Popper anchorEl={anchorEl} open={open}>...</Popper>
```

## Recharts Tooltips

Custom tooltip components receive empty `{}` props initially. Use `Partial<>`:

```tsx
function CustomTooltip({ active, payload }: Partial<TooltipContentProps<number, string>>) {
    if (!active || !payload?.length) return null
    // ...
}
```

## UI Components (MUI v5)

This project uses MUI v5 via `@mui/material`. Key patterns:

- Use `InputProps` (not `slotProps.input`) for TextField adornments
- Use `SelectProps` for Select customization
- Use `inputProps` (lowercase `i`) for native input HTML attributes
- Always reference MUI v5 documentation -- current MUI docs default to v6 syntax

```tsx
// GOOD: MUI v5 TextField with adornment
<TextField
    InputProps={{
        startAdornment: <InputAdornment position="start"><SearchIcon /></InputAdornment>
    }}
/>

// BAD: MUI v6 syntax -- does not work with MUI v5
<TextField slotProps={{ input: { startAdornment: ... } }} />
```

For Tooltip wrapping disabled buttons, add a `<span>` wrapper:

```tsx
<Tooltip title="Refresh">
    <span>
        <IconButton disabled={loading}><RefreshIcon /></IconButton>
    </span>
</Tooltip>
```

## Testing

Vitest + Testing Library. Prefer `mergeConfig(viteConfig, defineConfig(...))` so Vite
plugins and defines are inherited:

```ts
// vitest.config.ts
import { mergeConfig, defineConfig } from 'vitest/config'
import viteConfig from './vite.config'

export default mergeConfig(viteConfig, defineConfig({
    test: { environment: 'jsdom', globals: true, setupFiles: './src/test-setup.ts' }
}))
```
