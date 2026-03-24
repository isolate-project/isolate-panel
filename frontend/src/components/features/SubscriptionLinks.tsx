import { useState } from 'preact/hooks'
import { Modal } from '../ui/Modal'
import { Button } from '../ui/Button'
import { Spinner } from '../ui/Spinner'
import { Badge } from '../ui/Badge'
import { subscriptionApi } from '../../api/endpoints'
import type { User } from '../../types'
import { Copy, ExternalLink, Check } from 'lucide-preact'
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
  ]

  const handleCopy = async (key: string, text: string) => {
    try {
      await navigator.clipboard.writeText(text)
      setCopiedKey(key)
      setTimeout(() => setCopiedKey(null), 2000)
    } catch {
      // Fallback for older browsers
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
      // Error handled silently — user can retry
    } finally {
      setShortUrlLoading(false)
    }
  }

  if (!token) {
    return (
      <Modal isOpen={isOpen} onClose={onClose} title={t('subscriptions.title')}>
        <p className="text-secondary py-4">{t('subscriptions.noToken')}</p>
        <div className="flex justify-end">
          <Button variant="secondary" onClick={onClose}>
            {t('common.close')}
          </Button>
        </div>
      </Modal>
    )
  }

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={t('subscriptions.title')} size="lg">
      <p className="text-sm text-secondary mb-4">{t('subscriptions.description')}</p>

      <div className="space-y-3">
        {links.map((link) => (
          <div key={link.key} className="flex items-center justify-between p-3 border border-primary rounded-lg">
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 mb-1">
                <Badge variant="info">{link.label}</Badge>
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
                variant="secondary"
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
        <Button variant="secondary" onClick={onClose}>
          {t('common.close')}
        </Button>
      </div>
    </Modal>
  )
}
