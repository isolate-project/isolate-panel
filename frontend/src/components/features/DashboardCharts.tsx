import { useEffect, useState } from 'preact/hooks'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  ArcElement,
  Title,
  Tooltip,
  Legend,
  Filler,
} from 'chart.js'
import { Line, Bar } from 'react-chartjs-2'
import { statsApi } from '../../api/endpoints'
import { Card, CardContent, CardHeader, CardTitle } from '../ui/Card'
import { Skeleton } from '../ui/Skeleton'
import { BarChart3 } from 'lucide-preact'

// Register Chart.js components
ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  ArcElement,
  Title,
  Tooltip,
  Legend,
  Filler
)

interface TrafficPoint {
  date: string
  upload: number
  download: number
  total: number
}

interface TrafficOverviewResponse {
  days: number
  granularity: string
  points: TrafficPoint[]
  total_upload: number
  total_download: number
  total: number
}

interface TopUser {
  user_id: number
  username: string
  traffic_used_bytes: number
  traffic_limit_bytes: number | null
  is_active: boolean
}

const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return (bytes / Math.pow(k, i)).toFixed(1) + ' ' + sizes[i]
}

const formatShortDate = (dateStr: string): string => {
  const d = new Date(dateStr)
  return `${d.getMonth() + 1}/${d.getDate()}`
}

export function TrafficChart({ days = 7 }: { days?: number }) {
  const [data, setData] = useState<TrafficOverviewResponse | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    statsApi.trafficOverview({ days, granularity: 'daily' })
      .then(res => setData(res.data))
      .catch(() => setData(null))
      .finally(() => setLoading(false))
  }, [days])

  if (loading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <BarChart3 className="w-5 h-5 text-text-secondary" />
            Traffic Overview
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Skeleton className="h-64 w-full" />
        </CardContent>
      </Card>
    )
  }

  const points = data?.points || []
  const labels = points.map(p => formatShortDate(p.date))

  const chartData = {
    labels,
    datasets: [
      {
        label: 'Upload',
        data: points.map(p => p.upload),
        borderColor: 'rgb(99, 102, 241)',
        backgroundColor: 'rgba(99, 102, 241, 0.1)',
        fill: true,
        tension: 0.4,
        pointRadius: 3,
        pointHoverRadius: 6,
      },
      {
        label: 'Download',
        data: points.map(p => p.download),
        borderColor: 'rgb(16, 185, 129)',
        backgroundColor: 'rgba(16, 185, 129, 0.1)',
        fill: true,
        tension: 0.4,
        pointRadius: 3,
        pointHoverRadius: 6,
      },
    ],
  }

  const options = {
    responsive: true,
    maintainAspectRatio: false,
    interaction: {
      mode: 'index' as const,
      intersect: false,
    },
    plugins: {
      legend: {
        position: 'top' as const,
        labels: {
          usePointStyle: true,
          padding: 16,
          font: { size: 12 },
        },
      },
      tooltip: {
        backgroundColor: 'rgba(0, 0, 0, 0.8)',
        titleFont: { size: 13 },
        bodyFont: { size: 12 },
        padding: 12,
        cornerRadius: 8,
        callbacks: {
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          label: (ctx: any) =>
            `${ctx.dataset.label || ''}: ${formatBytes(ctx.parsed.y || 0)}`,
        },
      },
    },
    scales: {
      y: {
        beginAtZero: true,
        grid: {
          color: 'rgba(128, 128, 128, 0.1)',
        },
        ticks: {
          callback: (value: string | number) => formatBytes(Number(value)),
          font: { size: 11 },
        },
      },
      x: {
        grid: {
          display: false,
        },
        ticks: {
          font: { size: 11 },
        },
      },
    },
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-2">
            <BarChart3 className="w-5 h-5 text-text-secondary" />
            Traffic Overview
          </CardTitle>
          <div className="flex items-center gap-3 text-xs text-text-tertiary">
            <span>Total: {formatBytes(data?.total || 0)}</span>
            <span>↑ {formatBytes(data?.total_upload || 0)}</span>
            <span>↓ {formatBytes(data?.total_download || 0)}</span>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        {points.length > 0 ? (
          <div style={{ height: '260px' }}>
            <Line data={chartData} options={options} />
          </div>
        ) : (
          <div className="flex items-center justify-center h-64 text-text-tertiary text-sm">
            No traffic data available yet
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export function TopUsersChart({ limit = 5 }: { limit?: number }) {
  const [users, setUsers] = useState<TopUser[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    statsApi.topUsers({ limit })
      .then(res => setUsers(res.data?.users || []))
      .catch(() => setUsers([]))
      .finally(() => setLoading(false))
  }, [limit])

  if (loading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Top Users by Traffic</CardTitle>
        </CardHeader>
        <CardContent>
          <Skeleton className="h-64 w-full" />
        </CardContent>
      </Card>
    )
  }

  if (users.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Top Users by Traffic</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center h-48 text-text-tertiary text-sm">
            No user traffic data available
          </div>
        </CardContent>
      </Card>
    )
  }

  const chartData = {
    labels: users.map(u => u.username),
    datasets: [
      {
        label: 'Traffic Used',
        data: users.map(u => u.traffic_used_bytes),
        backgroundColor: [
          'rgba(99, 102, 241, 0.7)',
          'rgba(16, 185, 129, 0.7)',
          'rgba(245, 158, 11, 0.7)',
          'rgba(239, 68, 68, 0.7)',
          'rgba(139, 92, 246, 0.7)',
          'rgba(14, 165, 233, 0.7)',
          'rgba(236, 72, 153, 0.7)',
        ],
        borderColor: [
          'rgb(99, 102, 241)',
          'rgb(16, 185, 129)',
          'rgb(245, 158, 11)',
          'rgb(239, 68, 68)',
          'rgb(139, 92, 246)',
          'rgb(14, 165, 233)',
          'rgb(236, 72, 153)',
        ],
        borderWidth: 1,
        borderRadius: 6,
      },
    ],
  }

  const options = {
    responsive: true,
    maintainAspectRatio: false,
    indexAxis: 'y' as const,
    plugins: {
      legend: {
        display: false,
      },
      tooltip: {
        backgroundColor: 'rgba(0, 0, 0, 0.8)',
        cornerRadius: 8,
        padding: 12,
        callbacks: {
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          label: (ctx: any) => formatBytes(ctx.parsed.x || 0),
        },
      },
    },
    scales: {
      x: {
        beginAtZero: true,
        grid: {
          color: 'rgba(128, 128, 128, 0.1)',
        },
        ticks: {
          callback: (value: string | number) => formatBytes(Number(value)),
          font: { size: 11 },
        },
      },
      y: {
        grid: { display: false },
        ticks: {
          font: { size: 12, weight: 'bold' as const },
        },
      },
    },
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Top Users by Traffic</CardTitle>
      </CardHeader>
      <CardContent>
        <div style={{ height: '260px' }}>
          <Bar data={chartData} options={options} />
        </div>
      </CardContent>
    </Card>
  )
}
