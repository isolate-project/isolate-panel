import { useState } from 'preact/hooks'
import { PageLayout } from '../components/layout/PageLayout'
import { PageHeader } from '../components/layout/PageHeader'
import { Card, CardContent } from '../components/ui/Card'
import { Button } from '../components/ui/Button'
import { Badge } from '../components/ui/Badge'
import { Modal } from '../components/ui/Modal'
import { Input } from '../components/ui/Input'
import { Spinner } from '../components/ui/Spinner'
import { useQuery } from '../hooks/useQuery'
import { useMutation } from '../hooks/useMutation'
import { certificateApi } from '../api/endpoints'
import type { Certificate } from '../types'
import { useToastStore } from '../stores/toastStore'
import { Plus, RefreshCw, Ban, Trash2, Upload, Shield, AlertTriangle } from 'lucide-preact'
import { useTranslation } from 'react-i18next'

export function Certificates() {
  const { t } = useTranslation()
  const { addToast } = useToastStore()
  const [showRequestModal, setShowRequestModal] = useState(false)
  const [showUploadModal, setShowUploadModal] = useState(false)
  const [requestWildcard, setRequestWildcard] = useState(false)
  const [requestDomain, setRequestDomain] = useState('')

  const { data: certsData, isLoading, refetch } = useQuery<{ certificates: Certificate[]; total: number }>(
    'certificates',
    () => certificateApi.list().then(res => res.data as { certificates: Certificate[]; total: number })
  )

  const requestMutation = useMutation(
    (data: { domain: string; is_wildcard: boolean }) => certificateApi.request(data).then(res => res.data),
    {
      onSuccess: () => {
        addToast({ type: 'success', message: t('certificates.requested') })
        refetch()
        setShowRequestModal(false)
        setRequestDomain('')
        setRequestWildcard(false)
      },
      onError: (error: Error) => {
        addToast({ type: 'error', message: error.message })
      },
    }
  )

  const renewMutation = useMutation(
    (id: number) => certificateApi.renew(id).then(res => res.data),
    {
      onSuccess: () => {
        addToast({ type: 'success', message: t('certificates.renewed') })
        refetch()
      },
      onError: (error: Error) => {
        addToast({ type: 'error', message: error.message })
      },
    }
  )

  const revokeMutation = useMutation(
    (id: number) => certificateApi.revoke(id).then(res => res.data),
    {
      onSuccess: () => {
        addToast({ type: 'success', message: t('certificates.revoked') })
        refetch()
      },
      onError: (error: Error) => {
        addToast({ type: 'error', message: error.message })
      },
    }
  )

  const deleteMutation = useMutation(
    (id: number) => certificateApi.delete(id).then(res => res.data),
    {
      onSuccess: () => {
        addToast({ type: 'success', message: t('certificates.deleted') })
        refetch()
      },
      onError: (error: Error) => {
        addToast({ type: 'error', message: error.message })
      },
    }
  )

  const handleRequest = () => {
    if (!requestDomain) {
      addToast({ type: 'error', message: t('validation.domainRequired') })
      return
    }
    requestMutation.mutate({ domain: requestDomain, is_wildcard: requestWildcard })
  }

  const getStatusBadge = (cert: Certificate) => {
    const statusMap: Record<string, { variant: 'success' | 'warning' | 'danger' | 'info'; label: string }> = {
      active: { variant: 'success', label: t('certificates.status.active') },
      expiring: { variant: 'warning', label: t('certificates.status.expiring') },
      expired: { variant: 'danger', label: t('certificates.status.expired') },
      revoked: { variant: 'danger', label: t('certificates.status.revoked') },
      pending: { variant: 'info', label: t('certificates.status.pending') },
    }
    const status = statusMap[cert.status] || { variant: 'info' as const, label: cert.status }
    return <Badge variant={status.variant}>{status.label}</Badge>
  }

  const getDaysUntilExpiry = (cert: Certificate) => {
    const days = Math.floor((new Date(cert.not_after).getTime() - Date.now()) / (1000 * 60 * 60 * 24))
    if (days < 0) return t('certificates.expired')
    if (days === 0) return t('certificates.expiresToday')
    if (days === 1) return t('certificates.expiresTomorrow')
    if (days <= 30) return t('certificates.expiresInDays', { days })
    return t('certificates.validForDays', { days })
  }

  return (
    <PageLayout>
      <PageHeader
        title={t('nav.certificates')}
        description={t('certificates.description')}
        actions={
          <>
            <Button variant="outline" onClick={() => setShowUploadModal(true)}>
              <Upload className="w-4 h-4 mr-2" />
              {t('certificates.upload')}
            </Button>
            <Button onClick={() => setShowRequestModal(true)}>
              <Plus className="w-4 h-4 mr-2" />
              {t('certificates.request')}
            </Button>
          </>
        }
      />

      <Card>
      <CardContent className="p-6">
        {isLoading ? (
          <div className="flex justify-center py-12">
            <Spinner size="lg" />
          </div>
        ) : certsData?.certificates.length === 0 ? (
          <div className="text-center py-12">
            <Shield className="w-16 h-16 mx-auto text-secondary mb-4" />
            <h3 className="text-lg font-medium text-primary mb-2">{t('certificates.noCertificates')}</h3>
            <p className="text-secondary mb-4">{t('certificates.noCertificatesDesc')}</p>
            <Button onClick={() => setShowRequestModal(true)}>
              <Plus className="w-4 h-4 mr-2" />
              {t('certificates.requestFirst')}
            </Button>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-primary">
                  <th className="text-left py-3 px-4 text-sm font-medium text-secondary">{t('certificates.domain')}</th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-secondary">{t('certificates.issuer')}</th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-secondary">{t('certificates.validity')}</th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-secondary">{t('common.status')}</th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-secondary">{t('common.actions')}</th>
                </tr>
              </thead>
              <tbody>
                {certsData?.certificates.map((cert: Certificate) => (
                  <tr key={cert.id} className="border-b border-hover hover:bg-hover/50">
                    <td className="py-3 px-4">
                      <div className="flex items-center gap-2">
                        {cert.is_wildcard && <span className="text-primary">*</span>}
                        <span className="font-medium text-primary">{cert.domain}</span>
                      </div>
                    </td>
                    <td className="py-3 px-4 text-secondary">{cert.issuer || 'Manual'}</td>
                    <td className="py-3 px-4">
                      <div className="text-sm text-primary">{getDaysUntilExpiry(cert)}</div>
                      <div className="text-xs text-tertiary">
                        {new Date(cert.not_before).toLocaleDateString()} - {new Date(cert.not_after).toLocaleDateString()}
                      </div>
                    </td>
                    <td className="py-3 px-4">{getStatusBadge(cert)}</td>
                    <td className="py-3 px-4">
                      <div className="flex items-center gap-1">
                        {cert.status === 'active' || cert.status === 'expiring' ? (
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => renewMutation.mutate(cert.id)}
                            disabled={renewMutation.isLoading}
                          >
                            <RefreshCw className="w-4 h-4" />
                          </Button>
                        ) : null}
                        {cert.status !== 'revoked' && (
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => {
                              if (confirm(t('certificates.confirmRevoke'))) {
                                revokeMutation.mutate(cert.id)
                              }
                            }}
                            disabled={revokeMutation.isLoading}
                          >
                            <Ban className="w-4 h-4" />
                          </Button>
                        )}
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => {
                            if (confirm(t('certificates.confirmDelete'))) {
                              deleteMutation.mutate(cert.id)
                            }
                          }}
                          disabled={deleteMutation.isLoading}
                        >
                          <Trash2 className="w-4 h-4" />
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
            </CardContent>
    </Card>

      {/* Request Certificate Modal */}
      <Modal
        isOpen={showRequestModal}
        onClose={() => setShowRequestModal(false)}
        title={t('certificates.request')}
        size="md"
      >
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-primary mb-2">{t('certificates.domain')}</label>
            <Input
              value={requestDomain}
              onChange={(e) => setRequestDomain((e.target as HTMLInputElement).value)}
              placeholder="example.com"
            />
          </div>

          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="wildcard"
              checked={requestWildcard}
              onChange={(e) => setRequestWildcard(e.currentTarget.checked)}
              className="w-4 h-4"
            />
            <label htmlFor="wildcard" className="text-sm text-primary">
              {t('certificates.wildcard')} (*.example.com)
            </label>
          </div>

          <div className="p-3 bg-info/10 border border-info rounded-lg">
            <div className="flex items-start gap-2">
              <AlertTriangle className="w-5 h-5 text-info mt-0.5" />
              <div className="text-sm text-secondary">
                <p className="font-medium text-primary mb-1">{t('certificates.dns01Notice')}</p>
                <p>{t('certificates.dns01Desc')}</p>
              </div>
            </div>
          </div>

          <div className="flex justify-end gap-2 pt-4">
            <Button variant="outline" onClick={() => setShowRequestModal(false)}>
              {t('common.cancel')}
            </Button>
            <Button onClick={handleRequest} disabled={requestMutation.isLoading}>
              {requestMutation.isLoading ? <Spinner size="sm" /> : t('certificates.request')}
            </Button>
          </div>
        </div>
      </Modal>

      {/* Upload Certificate Modal */}
      <Modal
        isOpen={showUploadModal}
        onClose={() => setShowUploadModal(false)}
        title={t('certificates.upload')}
        size="lg"
      >
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-primary mb-2">{t('certificates.domain')}</label>
            <Input placeholder="example.com" />
          </div>

          <div>
            <label className="block text-sm font-medium text-primary mb-2">{t('certificates.certificatePem')}</label>
            <textarea
              className="w-full px-3 py-2 bg-surface border border-primary rounded-lg text-primary font-mono text-sm h-32 resize-none focus:outline-none focus:ring-2 focus:ring-primary"
              placeholder="-----BEGIN CERTIFICATE-----..."
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-primary mb-2">{t('certificates.privateKeyPem')}</label>
            <textarea
              className="w-full px-3 py-2 bg-surface border border-primary rounded-lg text-primary font-mono text-sm h-32 resize-none focus:outline-none focus:ring-2 focus:ring-primary"
              placeholder="-----BEGIN PRIVATE KEY-----..."
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-primary mb-2">{t('certificates.issuerPem')} ({t('common.optional')})</label>
            <textarea
              className="w-full px-3 py-2 bg-surface border border-primary rounded-lg text-primary font-mono text-sm h-24 resize-none focus:outline-none focus:ring-2 focus:ring-primary"
              placeholder="-----BEGIN CERTIFICATE-----..."
            />
          </div>

          <div className="flex justify-end gap-2 pt-4">
            <Button variant="outline" onClick={() => setShowUploadModal(false)}>
              {t('common.cancel')}
            </Button>
            <Button>{t('certificates.upload')}</Button>
          </div>
        </div>
      </Modal>
    </PageLayout>
  )
}
