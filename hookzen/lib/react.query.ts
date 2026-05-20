import { QueryClient } from "@tanstack/react-query";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1, // Only retry once on failure
      staleTime: 5 * 60 * 1000, // 5 mins
      gcTime: 10 * 60 * 1000, // 10 mins
      refetchOnWindowFocus: false, // Disable refetch when window regains focus
    },
    mutations: {
      retry: 0, // No retry on failure for mutations
    },
  },
});
