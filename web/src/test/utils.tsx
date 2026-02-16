/* eslint-disable react-refresh/only-export-components */
import { ReactElement, ReactNode } from 'react'
import { render, RenderOptions } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, MemoryRouterProps } from 'react-router-dom'

interface WrapperProps {
  children: ReactNode
}

interface CustomRenderOptions extends Omit<RenderOptions, 'wrapper'> {
  routerProps?: MemoryRouterProps
}

function createWrapper(routerProps?: MemoryRouterProps) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return function Wrapper({ children }: WrapperProps) {
    return (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter {...routerProps}>{children}</MemoryRouter>
      </QueryClientProvider>
    )
  }
}

export function renderWithRouter(
  ui: ReactElement,
  { routerProps, ...renderOptions }: CustomRenderOptions = {}
) {
  return render(ui, {
    wrapper: createWrapper(routerProps),
    ...renderOptions,
  })
}

export * from '@testing-library/react'
export { renderWithRouter as render }
