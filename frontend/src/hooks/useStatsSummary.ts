import { useQuery } from './useQuery'
import { statsApi } from '../api/endpoints'

interface StatsSummaryData {
  total_users: number
  active_users: number
  total_traffic_bytes: number
  running_cores: number
  total_cores: number
  active_connections: number
}

const fetchStatsSummary = () => statsApi.summary().then((res) => res.data)

export function useStatsSummary() {
  const { data, isLoading, error } = useQuery<StatsSummaryData>(
    'stats-summary',
    fetchStatsSummary,
    {
      refetchInterval: 5000, // Poll every 5 seconds as fallback for WebSocket
    }
  )

  return {
    data,
    isLoading,
    error,
  }
}