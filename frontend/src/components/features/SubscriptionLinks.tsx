import { useState } from 'preact/hooks'
import { Modal } from '../ui/Modal'
import { Button } from '../ui/Button'
import { Spinner } from '../ui/Spinner'
import { Badge } from '../ui/Badge'
import { subscriptionApi } from '../../api/endpoints'
import { useSubscriptionStats, useRegenerateToken } from '../../hooks/useSubscriptionStats'
import type { User } from '../../types'
import { Copy, ExternalLink, Check, QrCode, RefreshCw, BarChart3 } from 'lucide-preact'
import { useTranslation } from 'react-i18next'

interface SubscriptionLinksProps {
  isOpen: boolean
  onClose: () => void
  user: User
}

export function SubscriptionLinks({ isOpen, onClose, user }: SubscriptionLinksProps) {
  const { t } = useTranslation()
  const [copiedKey, setCopiedKey] = useState<string | null>(null)
  const [shortUrl, setShortUrl] = useState<string | null>(null)
  const [shortUrlLoading, setShortUrlLoading] = useState(false)
  const [showQR, setShowQR] = useState(false)
  const [showStats, setShowStats] = useState(false)
  const [statsDays, setStatsDays] = useState(7)

  const { data: stats, refetch: refetchStats } = useSubscriptionStats(user.id, statsDays)
  const regenerateMutation = useRegenerateToken(() => {
    setShortUrl(null) // Reset short URL since it's now invalid
    setShowQR(false)
  })

  const baseUrl = window.location.origin
  const token = user.subscription_token

  const links = [
    {
      key: 'v2ray',
      label: t('subscriptions.v2rayLink'),
      url: `${baseUrl}/sub/${token}`,
    },
    {
      key: 'clash',
      label: t('subscriptions.clashLink'),
      url: `${baseUrl}/sub/${token}/clash`,
    },
    {
      key: 'singbox',
      label: t('subscriptions.singboxLink'),
      url: `${baseUrl}/sub/${token}/singbox`,
    },
    {
      key: 'isolate',
      label: t('subscriptions.isolateLink'),
      url: `${baseUrl}/sub/${token}/isolate`,
    },
  ]

  const handleCopy = async (key: string, text: string) => {
    try {
      await navigator.clipboard.writeText(text)
      setCopiedKey(key)
      setTimeout(() => setCopiedKey(null), 2000)
    } catch {
      const textarea = document.createElement('textarea')
      textarea.value = text
      document.body.appendChild(textarea)
      textarea.select()
      document.execCommand('copy')
      document.body.removeChild(textarea)
      setCopiedKey(key)
      setTimeout(() => setCopiedKey(null), 2000)
    }
  }

  const handleGenerateShortUrl = async () => {
    setShortUrlLoading(true)
    try {
      const res = await subscriptionApi.getShortURL(user.id, token)
      const data = res.data as { short_url?: string; short_code?: string }
      if (data.short_url) {
        setShortUrl(data.short_url)
      } else if (data.short_code) {
        setShortUrl(`${baseUrl}/s/${data.short_code}`)
      }
    } catch {
      // Error handled silently
    } finally {
      setShortUrlLoading(false)
    }
  }

  const handleRegenerate = async () => {
    if (!confirm(t('subscriptions.confirmRegenerate'))) {
      return
    }
    try {
      await regenerateMutation.mutate({ userId: user.id })
    } catch {
      // Error handled by mutation onError
    }
  }

  const handleShowStats = () => {
    setShowStats(true)
    refetchStats()
  }

  if (!token) {
    return (
      <Modal isOpen={isOpen} onClose={onClose} title={t('subscriptions.title')}>
        <p className="text-secondary py-4">{t('subscriptions.noToken')}</p>
        <div className="flex justify-end">
          <Button variant="outline" onClick={onClose}>
            {t('common.close')}
          </Button>
        </div>
      </Modal>
    )
  }

  return (
    <>
      <Modal isOpen={isOpen} onClose={onClose} title={t('subscriptions.title')} size="lg">
        <p className="text-sm text-secondary mb-4">{t('subscriptions.description')}</p>

        {/* Action buttons */}
        <div className="flex gap-2 mb-4">
          <Button variant="outline" size="sm" onClick={() => setShowQR(true)}>
            <QrCode className="w-4 h-4 mr-1" />
            {t('subscriptions.showQR')}
          </Button>
          <Button variant="outline" size="sm" onClick={handleShowStats}>
            <BarChart3 className="w-4 h-4 mr-1" />
            {t('subscriptions.viewStats')}
          </Button>
          <Button
            variant="destructive"
            size="sm"
            onClick={handleRegenerate}
            disabled={regenerateMutation.isLoading}
          >
            <RefreshCw className="w-4 h-4 mr-1" />
            {t('subscriptions.regenerate')}
          </Button>
        </div>

        <div className="space-y-3">
          {links.map((link) => (
            <div key={link.key} className="flex items-center justify-between p-3 border border-primary rounded-lg">
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <Badge variant="outline">{link.label}</Badge>
                </div>
                <p className="text-xs text-tertiary truncate font-mono">{link.url}</p>
              </div>
              <div className="flex items-center gap-1 ml-2 shrink-0">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleCopy(link.key, link.url)}
                >
                  {copiedKey === link.key ? (
                    <Check className="w-4 h-4 text-green-500" />
                  ) : (
                    <Copy className="w-4 h-4" />
                  )}
                </Button>
                <a
                  href={link.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="p-1 hover:bg-hover rounded transition-base inline-flex"
                >
                  <ExternalLink className="w-4 h-4 text-secondary" />
                </a>
              </div>
            </div>
          ))}

          {/* Short URL */}
          <div className="border-t border-primary pt-3 mt-3">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium text-primary">{t('subscriptions.shortUrl')}</span>
              {!shortUrl && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleGenerateShortUrl}
                  disabled={shortUrlLoading}
                >
                  {shortUrlLoading ? <Spinner size="sm" /> : t('subscriptions.generateShortUrl')}
                </Button>
              )}
            </div>
            {shortUrl && (
              <div className="flex items-center justify-between p-3 mt-2 border border-primary rounded-lg">
                <p className="text-xs text-tertiary truncate font-mono flex-1">{shortUrl}</p>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleCopy('short', shortUrl)}
                >
                  {copiedKey === 'short' ? (
                    <Check className="w-4 h-4 text-green-500" />
                  ) : (
                    <Copy className="w-4 h-4" />
                  )}
                </Button>
              </div>
            )}
          </div>
        </div>

        <div className="flex justify-end pt-4">
          <Button variant="outline" onClick={onClose}>
            {t('common.close')}
          </Button>
        </div>
      </Modal>

      {/* QR Code Modal */}
      {showQR && (
        <Modal isOpen={showQR} onClose={() => setShowQR(false)} title={t('subscriptions.qrCode')} size="md">
          <div className="flex flex-col items-center py-4">
            <img
              src={`${baseUrl}/sub/${token}/qr`}
              alt="Subscription QR Code"
              className="w-64 h-64 object-contain"
            />
            <p className="text-sm text-secondary mt-4 text-center">
              {t('subscriptions.qrDescription')}
            </p>
          </div>
          <div className="flex justify-center">
            <Button variant="outline" onClick={() => setShowQR(false)}>
              {t('common.close')}
            </Button>
          </div>
        </Modal>
      )}

      {/* Stats Modal */}
      {showStats && (
        <Modal isOpen={showStats} onClose={() => setShowStats(false)} title={t('subscriptions.stats')} size="lg">
          <div className="space-y-4">
            {/* Days selector */}
            <div className="flex gap-2">
              {[7, 30, 90].map((d) => (
                <Button
                  key={d}
                  variant={statsDays === d ? 'primary' : 'secondary'}
                  size="sm"
                  onClick={() => setStatsDays(d)}
                >
                  {d} {t('common.days')}
                </Button>
              ))}
            </div>

            {stats ? (
              <>
                {/* Summary */}
                <div className="grid grid-cols-2 gap-3">
                  <div className="p-3 border border-primary rounded-lg">
                    <p className="text-xs text-secondary">{t('subscriptions.totalAccesses')}</p>
                    <p className="text-2xl font-bold text-primary">{stats.total_accesses}</p>
                  </div>
                  <div className="p-3 border border-primary rounded-lg">
                    <p className="text-xs text-secondary">{t('subscriptions.uniqueIPs')}</p>
                    <p className="text-2xl font-bold text-primary">{stats.unique_ips}</p>
                  </div>
                </div>

                {/* By format */}
                <div>
                  <h4 className="text-sm font-medium text-primary mb-2">{t('subscriptions.byFormat')}</h4>
                  <div className="space-y-2">
                    {Object.entries(stats.by_format).map(([format, count]) => (
                      <div key={format} className="flex items-center justify-between">
                        <span className="text-sm text-secondary capitalize">{format}</span>
                        <Badge variant="outline">{String(count)}</Badge>
                      </div>
                    ))}
                  </div>
                </div>

                {/* By day - CSS bar chart */}
                <div>
                  <h4 className="text-sm font-medium text-primary mb-2">{t('subscriptions.byDay')}</h4>
                  <div className="space-y-1">
                    {Object.entries(stats.by_day)
                      .sort(([a], [b]) => a.localeCompare(b))
                      .map(([day, count]) => {
                        const counts = Object.values(stats.by_day) as number[]
                        const maxCount = Math.max(...counts)
                        const width = maxCount > 0 ? ((count as number) / maxCount) * 100 : 0
                        return (
                          <div key={day} className="flex items-center gap-2">
                            <span className="text-xs text-secondary w-24 shrink-0">{day}</span>
                            <div className="flex-1 h-4 bg-surface rounded overflow-hidden">
                              <div
                                className="h-full bg-primary transition-all"
                                style={{ width: `${width}%` }}
                              />
                            </div>
                            <span className="text-xs text-tertiary w-8 text-right">{String(count)}</span>
                          </div>
                        )
                      })}
                  </div>
                </div>

                {/* Last access */}
                {stats.last_access && (
                  <div className="pt-2 border-t border-primary">
                    <p className="text-xs text-secondary">
                      {t('subscriptions.lastAccess')}: {new Date(stats.last_access).toLocaleString()}
                    </p>
                  </div>
                )}
              </>
            ) : (
              <div className="flex justify-center py-8">
                <Spinner size="md" />
              </div>
            )}
          </div>

          <div className="flex justify-end pt-4">
            <Button variant="outline" onClick={() => setShowStats(false)}>
              {t('common.close')}
            </Button>
          </div>
        </Modal>
      )}
    </>
  )
}
