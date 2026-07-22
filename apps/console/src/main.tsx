import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import {
  MutationCache,
  QueryClient,
  QueryClientProvider,
} from '@tanstack/react-query';
import { RouterProvider } from 'react-router-dom';
import './index.css';
import { router } from './router';
import { Toaster, toast } from './components/toast';
import { friendlyError } from './api/client';
import { applyTheme, useThemeStore } from './stores/theme';

// Apply persisted theme before first paint.
applyTheme(useThemeStore.getState().mode);

// Keep "system" mode reactive to OS changes.
window
  .matchMedia('(prefers-color-scheme: light)')
  .addEventListener('change', () => {
    if (useThemeStore.getState().mode === 'system') {
      applyTheme('system');
    }
  });

const queryClient = new QueryClient({
  // Any mutation that doesn't define its own onError surfaces a toast, so no
  // create/save/delete fails silently. Mutations with their own onError opt out.
  mutationCache: new MutationCache({
    onError: (error, _vars, _ctx, mutation) => {
      if (!mutation.options.onError) toast.error(friendlyError(error));
    },
  }),
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
      staleTime: 30_000,
    },
  },
});

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
      <Toaster />
    </QueryClientProvider>
  </StrictMode>,
);
