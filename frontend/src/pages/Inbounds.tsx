import { useState } from 'preact/hooks'
import { route } from 'preact-router'
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
import { useInbounds, useDeleteInbound, useAssignUser, useUnassignUser } from '../hooks/useInbounds'
import { useUsers } from '../hooks/useUsers'
import type { Inbound, User } from '../types'
import { Plus, Edit, Trash2, Users as UsersIcon, Search, UserMinus } from 'lucide-preact'
import { useTranslation } from 'react-i18next'

export function Inbounds() {
  const { t } = useTranslation()
  const { data: inbounds, isLoading, refetch } = useInbounds()
  const { mutate: deleteInbound } = useDeleteInbound()
  const { data: usersResponse } = useUsers()
  const { mutate: assignUser } = useAssignUser()
  const { mutate: unassignUser } = useUnassignUser()

  const [searchTerm, setSearchTerm] = useState('')
  const [protocolFilter, setProtocolFilter] = useState<string>('all')
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false)
  const [inboundToDelete, setInboundToDelete] = useState<Inbound | null>(null)
  const [isUsersModalOpen, setIsUsersModalOpen] = useState(false)
  const [inboundForUsers, setInboundForUsers] = useState<Inbound | null>(null)

  const allInbounds: Inbound[] = Array.isArray(inbounds) ? inbounds : []
  const allUsers: User[] = Array.isArray(usersResponse?.users) ? usersResponse.users : (Array.isArray(usersResponse) ? usersResponse as unknown as User[] : [])

  const handleDelete = async () => {
    if (inboundToDelete) {
      await deleteInbound(inboundToDelete.id)
      setIsDeleteModalOpen(false)
      setInboundToDelete(null)
      refetch()
    }
  }

  const handleManageUsers = (inbound: Inbound) => {
    setInboundForUsers(inbound)
    setIsUsersModalOpen(true)
  }

  const handleSearchChange = (e: Event) => {
    const target = e.target as HTMLInputElement
    setSearchTerm(target.value)
  }

  const handleProtocolFilterChange = (e: Event) => {
    const target = e.target as HTMLSelectElement
    setProtocolFilter(target.value)
  }

  const filteredInbounds = allInbounds.filter((inbound) => {
    const matchesSearch = searchTerm
      ? inbound.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
        inbound.protocol.toLowerCase().includes(searchTerm.toLowerCase())
      : true
    const matchesProtocol = protocolFilter === 'all' || inbound.protocol === protocolFilter
    return matchesSearch && matchesProtocol
  })

  return (
    <PageLayout>
      <PageHeader
        title={t('inbounds.title')}
        description={t('inbounds.description')}
        actions={
          <Button
            variant="primary"
            onClick={() => route('/inbounds/create')}
          >
            <Plus className="w-4 h-4 mr-2" />
            {t('inbounds.addInbound')}
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
                placeholder={t('inbounds.searchPlaceholder')}
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
                { value: 'vless', label: 'VLESS' },
                { value: 'vmess', label: 'VMess' },
                { value: 'trojan', label: 'Trojan' },
                { value: 'shadowsocks', label: 'Shadowsocks' },
                { value: 'hysteria2', label: 'Hysteria2' },
                { value: 'tuic', label: 'TUIC' },
              ]}
            />
          </div>
        </div>
      </Card>

      {isLoading ? (
        <Card className="flex items-center justify-center py-12">
          <Spinner size="lg" />
        </Card>
      ) : filteredInbounds.length > 0 ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {filteredInbounds.map((inbound) => (
            <Card key={inbound.id} className="p-6">
              <div className="flex items-start justify-between mb-4">
                <div>
                  <h3 className="text-lg font-semibold text-primary mb-1">
                    {inbound.name}
                  </h3>
                  <p className="text-sm text-tertiary">
                    {inbound.protocol.toUpperCase()} &bull; {t('inbounds.port')} {inbound.port}
                  </p>
                </div>
                <Badge variant={inbound.is_enabled ? 'success' : 'default'}>
                  {inbound.is_enabled ? t('common.active') : t('common.inactive')}
                </Badge>
              </div>

              <div className="space-y-2 mb-4">
                <div className="flex justify-between text-sm">
                  <span className="text-secondary">{t('inbounds.core')}:</span>
                  <span className="text-primary font-medium">{inbound.core?.name || '-'}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-secondary">{t('inbounds.listenAddress')}:</span>
                  <span className="text-primary font-medium">{inbound.listen_address}</span>
                </div>
                <div className="flex items-center gap-2 pt-1">
                  {inbound.tls_enabled && (
                    <Badge variant="info" className="text-xs">TLS</Badge>
                  )}
                  {inbound.reality_enabled && (
                    <Badge variant="info" className="text-xs">Reality</Badge>
                  )}
                </div>
              </div>

              <div className="flex gap-2">
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => handleManageUsers(inbound)}
                  className="flex-1"
                >
                  <UsersIcon className="w-4 h-4 mr-1" />
                  {t('inbounds.assignUsers')}
                </Button>
                <button
                  onClick={() => route(`/inbounds/${inbound.id}`)}
                  className="p-2 hover:bg-hover rounded transition-base"
                >
                  <Edit className="w-4 h-4 text-secondary" />
                </button>
                <button
                  onClick={() => {
                    setInboundToDelete(inbound)
                    setIsDeleteModalOpen(true)
                  }}
                  className="p-2 hover:bg-hover rounded transition-base"
                >
                  <Trash2 className="w-4 h-4 text-danger" />
                </button>
              </div>
            </Card>
          ))}
        </div>
      ) : (
        <Card className="text-center py-12">
          <p className="text-secondary mb-4">
            {searchTerm || protocolFilter !== 'all'
              ? t('inbounds.noMatchingInbounds')
              : t('inbounds.noInbounds')}
          </p>
          {(searchTerm || protocolFilter !== 'all') ? (
            <Button variant="secondary" onClick={() => {
              setSearchTerm('')
              setProtocolFilter('all')
            }}>
              {t('common.clearFilters')}
            </Button>
          ) : (
            <Button variant="primary" onClick={() => route('/inbounds/create')}>
              <Plus className="w-4 h-4 mr-2" />
              {t('inbounds.addInbound')}
            </Button>
          )}
        </Card>
      )}

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={isDeleteModalOpen}
        onClose={() => {
          setIsDeleteModalOpen(false)
          setInboundToDelete(null)
        }}
        title={t('inbounds.deleteInbound')}
      >
        <Alert variant="danger" className="mb-4">
          {t('inbounds.deleteConfirm')}
        </Alert>
        {inboundToDelete && (
          <p className="text-secondary mb-4">
            {t('inbounds.name')}: <strong>{inboundToDelete.name}</strong>
          </p>
        )}
        <div className="flex gap-3 justify-end">
          <Button
            variant="secondary"
            onClick={() => {
              setIsDeleteModalOpen(false)
              setInboundToDelete(null)
            }}
          >
            {t('common.cancel')}
          </Button>
          <Button variant="danger" onClick={handleDelete}>
            {t('common.delete')}
          </Button>
        </div>
      </Modal>

      {/* Manage Users Modal */}
      <Modal
        isOpen={isUsersModalOpen}
        onClose={() => {
          setIsUsersModalOpen(false)
          setInboundForUsers(null)
        }}
        title={`${t('inbounds.assignUsers')} - ${inboundForUsers?.name || ''}`}
        size="lg"
      >
        {inboundForUsers && (
          <InboundUsersManager
            inboundId={inboundForUsers.id}
            allUsers={allUsers}
            assignUser={assignUser}
            unassignUser={unassignUser}
            onDone={() => {
              setIsUsersModalOpen(false)
              setInboundForUsers(null)
              refetch()
            }}
          />
        )}
      </Modal>
    </PageLayout>
  )
}

// User assignment component
function InboundUsersManager({
  inboundId,
  allUsers,
  assignUser,
  unassignUser,
  onDone,
}: {
  inboundId: number
  allUsers: User[]
  assignUser: (data: { inboundId: number; userId: number }) => Promise<unknown>
  unassignUser: (data: { inboundId: number; userId: number }) => Promise<unknown>
  onDone: () => void
}) {
  const { t } = useTranslation()
  const [assigning, setAssigning] = useState<number | null>(null)
  const [unassigning, setUnassigning] = useState<number | null>(null)

  const handleAssign = async (userId: number) => {
    setAssigning(userId)
    try {
      await assignUser({ inboundId, userId })
    } finally {
      setAssigning(null)
    }
  }

  const handleUnassign = async (userId: number) => {
    setUnassigning(userId)
    try {
      await unassignUser({ inboundId, userId })
    } finally {
      setUnassigning(null)
    }
  }

  return (
    <div className="space-y-4">
      <p className="text-sm text-secondary">
        {t('inbounds.assignUsersDescription')}
      </p>

      {allUsers.length === 0 ? (
        <div className="text-center py-8 text-secondary">
          {t('inbounds.noUsersAvailable')}
        </div>
      ) : (
        <div className="space-y-2 max-h-80 overflow-y-auto">
          {allUsers.map((user) => (
            <div key={user.id} className="flex items-center justify-between p-3 border border-primary rounded-lg">
              <div>
                <div className="font-medium text-primary">{user.username}</div>
                <div className="text-xs text-tertiary">{user.email || '-'}</div>
              </div>
              <div className="flex items-center gap-2">
                <Badge variant={user.is_active ? 'success' : 'default'}>
                  {user.is_active ? t('common.active') : t('common.inactive')}
                </Badge>
                <Button
                  variant="primary"
                  size="sm"
                  onClick={() => handleAssign(user.id)}
                  disabled={assigning === user.id}
                >
                  {assigning === user.id ? <Spinner size="sm" /> : t('inbounds.assign')}
                </Button>
                <Button
                  variant="danger"
                  size="sm"
                  onClick={() => handleUnassign(user.id)}
                  disabled={unassigning === user.id}
                >
                  {unassigning === user.id ? <Spinner size="sm" /> : (
                    <>
                      <UserMinus className="w-3 h-3 mr-1" />
                      {t('inbounds.unassign')}
                    </>
                  )}
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      <div className="flex justify-end pt-2">
        <Button variant="secondary" onClick={onDone}>
          {t('common.close')}
        </Button>
      </div>
    </div>
  )
}
