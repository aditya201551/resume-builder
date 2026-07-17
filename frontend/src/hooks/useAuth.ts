import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { getMe, logout as logoutRequest } from '@/lib/api'

export const authQueryKey = ['auth', 'me'] as const

export function useAuth() {
  const query = useQuery({
    queryKey: authQueryKey,
    queryFn: getMe,
    staleTime: 5 * 60 * 1000,
  })

  const queryClient = useQueryClient()
  const logoutMutation = useMutation({
    mutationFn: logoutRequest,
    onSuccess: () => {
      queryClient.setQueryData(authQueryKey, null)
    },
  })

  return {
    user: query.data ?? null,
    isLoading: query.isLoading,
    isError: query.isError,
    logout: logoutMutation.mutate,
  }
}
