import { useState } from 'preact/hooks'
import { route } from 'preact-router'

import { PageLayout } from '../components/layout/PageLayout'
import { PageHeader } from '../components/layout/PageHeader'
import { Card, CardContent } from '../components/ui/Card'
import { Button } from '../components/ui/Button'
import { Badge } from '../components/ui/Badge'
import { Spinner } from '../components/ui/Spinner'
import { Modal } from '../components/ui/Modal'
import { Alert } from '../components/ui/Alert'
import { useInbound, useDeleteInbound } from '../hooks/useInbounds'
import { useInboundUsers, useBulkAssignUsers } from '../hooks/useInboundUsers'
import { useUsers } from '../hooks/useUsers'
import type { User } from '../types'
import { ArrowLeft, Edit, Trash2, Users as UsersIcon, UserPlus, UserMinus, Shield, Globe } from 'lucide-preact'
import { useTranslation } from 'react-i18next'

type Tab = 'overview' | 'users' | 'config'

export function InboundDetail({ id }: { id: number }) {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState<Tab>('overview')
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false)
  const [isAddUsersModalOpen, setIsAddUsersModalOpen] = useState(false)

  const { data: inbound, isLoading } = useInbound(id)
  const { data: inboundUsersData, refetch: refetchUsers } = useInboundUsers(id)
  const { data: allUsersResponse } = useUsers()
  const { mutate: deleteInbound } = useDeleteInbound()
  const { mutate: bulkAssign } = useBulkAssignUsers()

  const assignedUsers: User[] = inboundUsersData?.users || []
  const allUsers: User[] = Array.isArray(allUsersResponse?.users)
    ? allUsersResponse.users
    : (Array.isArray(allUsersResponse) ? allUsersResponse as unknown as User[] : [])

  const unassignedUsers = allUsers.filter(
    (u) => !assignedUsers.some((au) => au.id === u.id)
  )

  const handleDelete = async () => {
    await deleteInbound(id)
    setIsDeleteModalOpen(false)
    route('/inbounds')
  }

  const handleAddUser = async (userId: number) => {
    await bulkAssign({ inboundId: id, addUserIds: [userId], removeUserIds: [] })
    refetchUsers()
  }

  const handleRemoveUser = async (userId: number) => {
    await bulkAssign({ inboundId: id, addUserIds: [], removeUserIds: [userId] })
    refetchUsers()
  }

  if (isLoading) {
    return (
      <PageLayout>
        <Card className="flex items-center justify-center py-12">
      <CardContent className="p-6">
          <Spinner size="lg" />
              </CardContent>
    </Card>
      </PageLayout>
    )
  }

  if (!inbound) {
    return (
      <PageLayout>
        <Card className="text-center py-12">
      <CardContent className="p-6">
          <p className="text-secondary mb-4">{t('errors.notFound')}</p>
          <Button variant="outline" onClick={() => route('/inbounds')}>
            {t('inboundDetail.backToInbounds')}
          </Button>
              </CardContent>
    </Card>
      </PageLayout>
    )
  }

  let configObj: Record<string, unknown> = {}
  try {
    configObj = inbound.config_json ? JSON.parse(inbound.config_json) : {}
  } catch {
    // ignore parse errors
  }

  const tabs: { key: Tab; label: string }[] = [
    { key: 'overview', label: t('inboundDetail.overview') },
    { key: 'users', label: t('inboundDetail.users') },
    { key: 'config', label: t('inboundDetail.config') },
  ]

  return (
    <PageLayout>
      <PageHeader
        title={inbound.name}
        description={`${inbound.protocol.toUpperCase()} - ${t('inbounds.port')} ${inbound.port}`}
        actions={
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => route('/inbounds')}>
              <ArrowLeft className="w-4 h-4 mr-1" />
              {t('inboundDetail.backToInbounds')}
            </Button>
            <Button variant="outline" onClick={() => route(`/inbounds/${id}/edit`)}>
              <Edit className="w-4 h-4 mr-1" />
              {t('inboundDetail.editInbound')}
            </Button>
            <Button variant="danger" onClick={() => setIsDeleteModalOpen(true)}>
              <Trash2 className="w-4 h-4 mr-1" />
              {t('common.delete')}
            </Button>
          </div>
        }
      />

      {/* Tabs */}
      <Card className="mb-6">
      <CardContent className="p-6">
        <div className="flex border-b border-primary">
          {tabs.map((tab) => (
            <button
              key={tab.key}
              className={`px-4 py-2 text-sm font-medium border-b-2 transition-base ${
                activeTab === tab.key
                  ? 'border-blue-500 text-primary'
                  : 'border-transparent text-secondary hover:text-primary'
              }`}
              onClick={() => setActiveTab(tab.key)}
            >
              {tab.label}
            </button>
          ))}
        </div>
            </CardContent>
    </Card>

      {/* Tab Content */}
      {activeTab === 'overview' && (
        <Card>
      <CardContent className="p-6">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="space-y-4">
              <div className="flex justify-between text-sm">
                <span className="text-secondary">{t('inboundDetail.protocol')}:</span>
                <Badge variant="outline">{inbound.protocol.toUpperCase()}</Badge>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-secondary">{t('inboundDetail.port')}:</span>
                <span className="text-primary font-medium">{inbound.port}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-secondary">{t('inboundDetail.listenAddress')}:</span>
                <span className="text-primary font-medium">{inbound.listen_address}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-secondary">{t('inboundDetail.core')}:</span>
                <span className="text-primary font-medium">{inbound.core?.name || '-'}</span>
              </div>
            </div>
            <div className="space-y-4">
              <div className="flex justify-between text-sm">
                <span className="text-secondary">{t('common.status')}:</span>
                <Badge variant={inbound.is_enabled ? 'success' : 'default'}>
                  {inbound.is_enabled ? t('inboundDetail.enabled') : t('inboundDetail.disabled')}
                </Badge>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-secondary">{t('inboundDetail.tls')}:</span>
                <div className="flex items-center gap-1">
                  {inbound.tls_enabled ? (
                    <><Shield className="w-4 h-4 text-green-500" /><span className="text-green-600">{t('inboundDetail.enabled')}</span></>
                  ) : (
                    <span className="text-tertiary">{t('inboundDetail.disabled')}</span>
                  )}
                </div>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-secondary">{t('inboundDetail.reality')}:</span>
                <div className="flex items-center gap-1">
                  {inbound.reality_enabled ? (
                    <><Globe className="w-4 h-4 text-blue-500" /><span className="text-blue-600">{t('inboundDetail.enabled')}</span></>
                  ) : (
                    <span className="text-tertiary">{t('inboundDetail.disabled')}</span>
                  )}
                </div>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-secondary">{t('inboundDetail.createdAt')}:</span>
                <span className="text-primary">{new Date(inbound.created_at).toLocaleDateString()}</span>
              </div>
            </div>
          </div>
              </CardContent>
    </Card>
      )}

      {activeTab === 'users' && (
        <Card>
      <CardContent className="p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-primary">
              <UsersIcon className="w-5 h-5 inline mr-2" />
              {t('inboundDetail.assignedUsers')} ({assignedUsers.length})
            </h3>
            <Button variant="default" size="sm" onClick={() => setIsAddUsersModalOpen(true)}>
              <UserPlus className="w-4 h-4 mr-1" />
              {t('inboundDetail.addUsers')}
            </Button>
          </div>

          {assignedUsers.length === 0 ? (
            <div className="text-center py-8 text-secondary">
              {t('inboundDetail.noUsers')}
            </div>
          ) : (
            <div className="space-y-2">
              {assignedUsers.map((user) => (
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
                      variant="danger"
                      size="sm"
                      onClick={() => handleRemoveUser(user.id)}
                    >
                      <UserMinus className="w-3 h-3 mr-1" />
                      {t('inboundDetail.removeUser')}
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}
              </CardContent>
    </Card>
      )}

      {activeTab === 'config' && (
        <Card>
      <CardContent className="p-6">
          <h3 className="text-lg font-semibold text-primary mb-4">{t('inboundDetail.configPreview')}</h3>
          <pre className="p-4 bg-secondary rounded-lg text-sm text-primary overflow-x-auto">
            {JSON.stringify(configObj, null, 2)}
          </pre>
              </CardContent>
    </Card>
      )}

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={isDeleteModalOpen}
        onClose={() => setIsDeleteModalOpen(false)}
        title={t('inbounds.deleteInbound')}
      >
        <Alert variant="danger" className="mb-4">
          {t('inbounds.deleteConfirm')}
        </Alert>
        <p className="text-secondary mb-4">
          {t('inbounds.name')}: <strong>{inbound.name}</strong>
        </p>
        <div className="flex gap-3 justify-end">
          <Button variant="outline" onClick={() => setIsDeleteModalOpen(false)}>
            {t('common.cancel')}
          </Button>
          <Button variant="danger" onClick={handleDelete}>
            {t('common.delete')}
          </Button>
        </div>
      </Modal>

      {/* Add Users Modal */}
      <Modal
        isOpen={isAddUsersModalOpen}
        onClose={() => setIsAddUsersModalOpen(false)}
        title={t('inboundDetail.addUsers')}
        size="lg"
      >
        {unassignedUsers.length === 0 ? (
          <div className="text-center py-8 text-secondary">
            {t('inbounds.noUsersAvailable')}
          </div>
        ) : (
          <div className="space-y-2 max-h-80 overflow-y-auto">
            {unassignedUsers.map((user) => (
              <div key={user.id} className="flex items-center justify-between p-3 border border-primary rounded-lg">
                <div>
                  <div className="font-medium text-primary">{user.username}</div>
                  <div className="text-xs text-tertiary">{user.email || '-'}</div>
                </div>
                <Button
                  variant="default"
                  size="sm"
                  onClick={() => handleAddUser(user.id)}
                >
                  <UserPlus className="w-3 h-3 mr-1" />
                  {t('inbounds.assign')}
                </Button>
              </div>
            ))}
          </div>
        )}
        <div className="flex justify-end pt-4">
          <Button variant="outline" onClick={() => setIsAddUsersModalOpen(false)}>
            {t('common.close')}
          </Button>
        </div>
      </Modal>
    </PageLayout>
  )
}
