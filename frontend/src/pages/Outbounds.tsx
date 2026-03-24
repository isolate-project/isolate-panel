import { useState } from 'preact/hooks'
import { PageLayout } from '../components/layout/PageLayout'
import { PageHeader } from '../components/layout/PageHeader'
import { Card } from '../components/ui/Card'
import { Button } from '../components/ui/Button'
import { Badge } from '../components/ui/Badge'
import { Spinner } from '../components/ui/Spinner'
import { Modal } from '../components/ui/Modal'
import { Alert } from '../components/ui/Alert'
import { Input } from '../components/ui/Input'
import { Select } from '../components/ui/Select'
import { useOutbounds, useDeleteOutbound } from '../hooks/useOutbounds'
import { OutboundForm } from '../components/forms/OutboundForm'
import type { Outbound } from '../types'
import { Plus, Edit, Trash2, Search } from 'lucide-preact'
import { useTranslation } from 'react-i18next'

export function Outbounds() {
  const { t } = useTranslation()
  const { data: outbounds, isLoading, refetch } = useOutbounds()
  const { mutate: deleteOutbound } = useDeleteOutbound()

  const [searchTerm, setSearchTerm] = useState('')
  const [protocolFilter, setProtocolFilter] = useState<string>('all')
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false)
  const [isEditModalOpen, setIsEditModalOpen] = useState(false)
  const [selectedOutbound, setSelectedOutbound] = useState<Outbound | null>(null)
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false)
  const [outboundToDelete, setOutboundToDelete] = useState<Outbound | null>(null)

  const allOutbounds: Outbound[] = Array.isArray(outbounds) ? outbounds : []

  const handleDelete = async () => {
    if (outboundToDelete) {
      await deleteOutbound(outboundToDelete.id)
      setIsDeleteModalOpen(false)
      setOutboundToDelete(null)
      refetch()
    }
  }

  const handleSearchChange = (e: Event) => {
    const target = e.target as HTMLInputElement
    setSearchTerm(target.value)
  }

  const handleProtocolFilterChange = (e: Event) => {
    const target = e.target as HTMLSelectElement
    setProtocolFilter(target.value)
  }

  const filteredOutbounds = allOutbounds.filter((outbound) => {
    const matchesSearch = searchTerm
      ? outbound.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
        outbound.protocol.toLowerCase().includes(searchTerm.toLowerCase())
      : true
    const matchesProtocol = protocolFilter === 'all' || outbound.protocol === protocolFilter
    return matchesSearch && matchesProtocol
  })

  const protocols = [...new Set(allOutbounds.map((o) => o.protocol))]

  return (
    <PageLayout>
      <PageHeader
        title={t('outbounds.title')}
        description={t('outbounds.description')}
        actions={
          <Button
            variant="primary"
            onClick={() => setIsCreateModalOpen(true)}
          >
            <Plus className="w-4 h-4 mr-2" />
            {t('outbounds.addOutbound')}
          </Button>
        }
      />

      {/* Search and Filter Bar */}
      <Card className="mb-4">
        <div className="flex flex-col md:flex-row gap-4">
          <div className="flex-1">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-tertiary" />
              <Input
                type="text"
                placeholder={t('outbounds.searchPlaceholder')}
                value={searchTerm}
                onChange={handleSearchChange}
                className="pl-10"
              />
            </div>
          </div>
          <div className="w-full md:w-48">
            <Select
              value={protocolFilter}
              onChange={handleProtocolFilterChange}
              options={[
                { value: 'all', label: t('common.all') },
                ...protocols.map((p) => ({ value: p, label: p.toUpperCase() })),
              ]}
            />
          </div>
        </div>
      </Card>

      {isLoading ? (
        <Card className="flex items-center justify-center py-12">
          <Spinner size="lg" />
        </Card>
      ) : filteredOutbounds.length > 0 ? (
        <Card padding="none">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-secondary border-b border-primary">
                <tr>
                  <th className="px-4 py-3 text-left text-sm font-medium text-primary">{t('outbounds.name')}</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-primary">{t('outbounds.protocol')}</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-primary">{t('outbounds.core')}</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-primary">{t('outbounds.priority')}</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-primary">{t('common.status')}</th>
                  <th className="px-4 py-3 text-right text-sm font-medium text-primary">{t('common.actions')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-primary">
                {filteredOutbounds.map((outbound) => (
                  <tr key={outbound.id} className="hover:bg-hover transition-base">
                    <td className="px-4 py-3">
                      <div className="font-medium text-primary">{outbound.name}</div>
                    </td>
                    <td className="px-4 py-3">
                      <Badge variant="info">{outbound.protocol.toUpperCase()}</Badge>
                    </td>
                    <td className="px-4 py-3 text-sm text-secondary">
                      {outbound.core?.name || '-'}
                    </td>
                    <td className="px-4 py-3 text-sm text-secondary">
                      {outbound.priority}
                    </td>
                    <td className="px-4 py-3">
                      <Badge variant={outbound.is_enabled ? 'success' : 'default'}>
                        {outbound.is_enabled ? t('common.active') : t('common.inactive')}
                      </Badge>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center justify-end gap-2">
                        <button
                          onClick={() => {
                            setSelectedOutbound(outbound)
                            setIsEditModalOpen(true)
                          }}
                          className="p-1 hover:bg-hover rounded transition-base"
                        >
                          <Edit className="w-4 h-4 text-secondary" />
                        </button>
                        <button
                          onClick={() => {
                            setOutboundToDelete(outbound)
                            setIsDeleteModalOpen(true)
                          }}
                          className="p-1 hover:bg-hover rounded transition-base"
                        >
                          <Trash2 className="w-4 h-4 text-danger" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      ) : (
        <Card className="text-center py-12">
          <p className="text-secondary mb-4">
            {searchTerm || protocolFilter !== 'all'
              ? t('outbounds.noMatchingOutbounds')
              : t('outbounds.noOutbounds')}
          </p>
          {searchTerm || protocolFilter !== 'all' ? (
            <Button variant="secondary" onClick={() => {
              setSearchTerm('')
              setProtocolFilter('all')
            }}>
              {t('common.clearFilters')}
            </Button>
          ) : (
            <Button variant="primary" onClick={() => setIsCreateModalOpen(true)}>
              <Plus className="w-4 h-4 mr-2" />
              {t('outbounds.addOutbound')}
            </Button>
          )}
        </Card>
      )}

      {/* Create Outbound Modal */}
      <Modal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        title={t('outbounds.addOutbound')}
        size="lg"
      >
        <OutboundForm
          onSuccess={() => {
            setIsCreateModalOpen(false)
            refetch()
          }}
          onCancel={() => setIsCreateModalOpen(false)}
        />
      </Modal>

      {/* Edit Outbound Modal */}
      <Modal
        isOpen={isEditModalOpen}
        onClose={() => {
          setIsEditModalOpen(false)
          setSelectedOutbound(null)
        }}
        title={t('outbounds.editOutbound')}
        size="lg"
      >
        {selectedOutbound && (
          <OutboundForm
            outbound={selectedOutbound}
            onSuccess={() => {
              setIsEditModalOpen(false)
              setSelectedOutbound(null)
              refetch()
            }}
            onCancel={() => {
              setIsEditModalOpen(false)
              setSelectedOutbound(null)
            }}
          />
        )}
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={isDeleteModalOpen}
        onClose={() => {
          setIsDeleteModalOpen(false)
          setOutboundToDelete(null)
        }}
        title={t('outbounds.deleteOutbound')}
      >
        <Alert variant="danger" className="mb-4">
          {t('outbounds.deleteConfirm')}
        </Alert>
        {outboundToDelete && (
          <p className="text-secondary mb-4">
            {t('outbounds.name')}: <strong>{outboundToDelete.name}</strong>
          </p>
        )}
        <div className="flex gap-3 justify-end">
          <Button
            variant="secondary"
            onClick={() => {
              setIsDeleteModalOpen(false)
              setOutboundToDelete(null)
            }}
          >
            {t('common.cancel')}
          </Button>
          <Button variant="danger" onClick={handleDelete}>
            {t('common.delete')}
          </Button>
        </div>
      </Modal>
    </PageLayout>
  )
}
