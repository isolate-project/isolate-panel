import { useState, useMemo } from 'preact/hooks'
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
import { useUsers, useDeleteUser, useRegenerateCredentials, useUserInbounds } from '../hooks/useUsers'
import { UserForm } from '../components/forms/UserForm'
import { SubscriptionLinks } from '../components/features/SubscriptionLinks'
import type { User, Inbound } from '../types'
import { Plus, Edit, Trash2, RefreshCw, Copy, List, ChevronLeft, ChevronRight, Search, Link as LinkIcon } from 'lucide-preact'
import { useTranslation } from 'react-i18next'

export function Users() {
  const { t } = useTranslation()
  
  // Pagination state
  const [page, setPage] = useState(1)
  const [limit, setLimit] = useState(20)
  
  // Search and filter state
  const [searchTerm, setSearchTerm] = useState('')
  const [statusFilter, setStatusFilter] = useState<'all' | 'active' | 'inactive'>('all')
  
  const { data: response, isLoading, refetch } = useUsers({ page, limit })
  const { mutate: deleteUser } = useDeleteUser()
  const { mutate: regenerateCredentials } = useRegenerateCredentials()

  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false)
  const [isEditModalOpen, setIsEditModalOpen] = useState(false)
  const [selectedUser, setSelectedUser] = useState<User | null>(null)
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false)
  const [userToDelete, setUserToDelete] = useState<User | null>(null)
  const [isInboundsModalOpen, setIsInboundsModalOpen] = useState(false)
  const [userForInbounds, setUserForInbounds] = useState<User | null>(null)
  const [isSubscriptionOpen, setIsSubscriptionOpen] = useState(false)
  const [userForSubscription, setUserForSubscription] = useState<User | null>(null)

  // Extract users and pagination info from response
  const users = response?.users || []
  const total = response?.total || 0
  const totalPages = Math.ceil(total / limit)

  // Filter users based on search and status
  const filteredUsers = useMemo(() => {
    let filtered = users

    // Search filter
    if (searchTerm) {
      const term = searchTerm.toLowerCase()
      filtered = filtered.filter((user: User) =>
        user.username?.toLowerCase().includes(term) ||
        user.email?.toLowerCase().includes(term) ||
        user.uuid?.toLowerCase().includes(term)
      )
    }

    // Status filter
    if (statusFilter !== 'all') {
      filtered = filtered.filter((user: User) =>
        statusFilter === 'active' ? user.is_active : !user.is_active
      )
    }

    return filtered
  }, [users, searchTerm, statusFilter])

  const handleDelete = async () => {
    if (userToDelete) {
      await deleteUser(userToDelete.id)
      setIsDeleteModalOpen(false)
      setUserToDelete(null)
      refetch()
    }
  }

  const handleRegenerate = async (userId: number) => {
    if (confirm(t('users.regenerateCredentials') + '?')) {
      await regenerateCredentials(userId)
      refetch()
    }
  }

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
  }

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
  }

  const handleViewInbounds = (user: User) => {
    setUserForInbounds(user)
    setIsInboundsModalOpen(true)
  }

  const handleSearchChange = (e: Event) => {
    const target = e.target as HTMLInputElement
    setSearchTerm(target.value)
  }

  const handleStatusFilterChange = (e: Event) => {
    const target = e.target as HTMLSelectElement
    setStatusFilter(target.value as 'all' | 'active' | 'inactive')
  }

  const handleLimitChange = (e: Event) => {
    const target = e.target as HTMLSelectElement
    setLimit(Number(target.value))
    setPage(1)
  }

  return (
    <PageLayout>
      <PageHeader
        title={t('users.title')}
        description={t('users.description')}
        actions={
          <Button
            variant="primary"
            onClick={() => setIsCreateModalOpen(true)}
          >
            <Plus className="w-4 h-4 mr-2" />
            {t('users.addUser')}
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
                placeholder={t('users.searchPlaceholder') || 'Search by username, email, or UUID...'}
                value={searchTerm}
                onChange={handleSearchChange}
                className="pl-10"
              />
            </div>
          </div>
          <div className="w-full md:w-48">
            <Select
              value={statusFilter}
              onChange={handleStatusFilterChange}
              options={[
                { value: 'all', label: t('common.all') || 'All Status' },
                { value: 'active', label: t('common.active') || 'Active' },
                { value: 'inactive', label: t('common.inactive') || 'Inactive' },
              ]}
            />
          </div>
          <div className="w-full md:w-32">
            <Select
              value={limit.toString()}
              onChange={handleLimitChange}
              options={[
                { value: '10', label: '10 / page' },
                { value: '20', label: '20 / page' },
                { value: '50', label: '50 / page' },
                { value: '100', label: '100 / page' },
              ]}
            />
          </div>
        </div>
      </Card>

      {isLoading ? (
        <Card className="flex items-center justify-center py-12">
          <Spinner size="lg" />
        </Card>
      ) : filteredUsers && filteredUsers.length > 0 ? (
        <>
          <Card padding="none">
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className="bg-secondary border-b border-primary">
                  <tr>
                    <th className="px-4 py-3 text-left text-sm font-medium text-primary">
                      {t('users.username')}
                    </th>
                    <th className="px-4 py-3 text-left text-sm font-medium text-primary">
                      {t('users.email')}
                    </th>
                    <th className="px-4 py-3 text-left text-sm font-medium text-primary">
                      {t('users.trafficUsed')}
                    </th>
                    <th className="px-4 py-3 text-left text-sm font-medium text-primary">
                      {t('common.status')}
                    </th>
                    <th className="px-4 py-3 text-left text-sm font-medium text-primary">
                      {t('users.createdAt')}
                    </th>
                    <th className="px-4 py-3 text-right text-sm font-medium text-primary">
                      {t('common.actions')}
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-primary">
                  {filteredUsers.map((user: User) => (
                    <tr key={user.id} className="hover:bg-hover transition-base">
                      <td className="px-4 py-3">
                        <div className="font-medium text-primary">{user.username}</div>
                        <div className="text-xs text-tertiary">UUID: {user.uuid?.substring(0, 8)}...</div>
                      </td>
                      <td className="px-4 py-3 text-sm text-secondary">
                        {user.email || '-'}
                      </td>
                      <td className="px-4 py-3 text-sm text-secondary">
                        {formatBytes(user.traffic_used_bytes || 0)}
                        {user.traffic_limit_bytes && (
                          <span className="text-tertiary">
                            {' / '}
                            {formatBytes(user.traffic_limit_bytes)}
                          </span>
                        )}
                      </td>
                      <td className="px-4 py-3">
                        <Badge variant={user.is_active ? 'success' : 'default'}>
                          {user.is_active ? t('common.active') : t('common.inactive')}
                        </Badge>
                      </td>
                      <td className="px-4 py-3 text-sm text-secondary">
                        {new Date(user.created_at).toLocaleDateString()}
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex items-center justify-end gap-2">
                          <button
                            onClick={() => handleViewInbounds(user)}
                            className="p-1 hover:bg-hover rounded transition-base"
                            title={t('users.viewInbounds')}
                          >
                            <List className="w-4 h-4 text-secondary" />
                          </button>
                          <button
                            onClick={() => {
                              setUserForSubscription(user)
                              setIsSubscriptionOpen(true)
                            }}
                            className="p-1 hover:bg-hover rounded transition-base"
                            title={t('subscriptions.title')}
                          >
                            <LinkIcon className="w-4 h-4 text-secondary" />
                          </button>
                          <button
                            onClick={() => copyToClipboard(user.subscription_token)}
                            className="p-1 hover:bg-hover rounded transition-base"
                            title={t('users.copyToken')}
                          >
                            <Copy className="w-4 h-4 text-secondary" />
                          </button>
                          <button
                            onClick={() => handleRegenerate(user.id)}
                            className="p-1 hover:bg-hover rounded transition-base"
                            title={t('users.regenerateCredentials')}
                          >
                            <RefreshCw className="w-4 h-4 text-secondary" />
                          </button>
                          <button
                            onClick={() => {
                              setSelectedUser(user)
                              setIsEditModalOpen(true)
                            }}
                            className="p-1 hover:bg-hover rounded transition-base"
                            title={t('users.editUser')}
                          >
                            <Edit className="w-4 h-4 text-secondary" />
                          </button>
                          <button
                            onClick={() => {
                              setUserToDelete(user)
                              setIsDeleteModalOpen(true)
                            }}
                            className="p-1 hover:bg-hover rounded transition-base"
                            title={t('users.deleteUser')}
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

          {/* Pagination Controls */}
          <Card className="mt-4">
            <div className="flex items-center justify-between">
              <div className="text-sm text-secondary">
                {t('users.showingResults', {
                  from: ((page - 1) * limit) + 1,
                  to: Math.min(page * limit, total),
                  total: total,
                })}
              </div>
              <div className="flex items-center gap-2">
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => setPage(p => Math.max(1, p - 1))}
                  disabled={page === 1}
                >
                  <ChevronLeft className="w-4 h-4" />
                </Button>
                <span className="text-sm text-secondary">
                  {t('users.pageOf', { page, totalPages })}
                </span>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => setPage(p => Math.min(totalPages, p + 1))}
                  disabled={page === totalPages}
                >
                  <ChevronRight className="w-4 h-4" />
                </Button>
              </div>
            </div>
          </Card>
        </>
      ) : searchTerm || statusFilter !== 'all' ? (
        <Card className="text-center py-12">
          <p className="text-secondary mb-4">{t('users.noUsersFiltered')}</p>
          <Button variant="secondary" onClick={() => {
            setSearchTerm('')
            setStatusFilter('all')
          }}>
            {t('common.clearFilters')}
          </Button>
        </Card>
      ) : (
        <Card className="text-center py-12">
          <p className="text-secondary mb-4">{t('users.noUsersYet')}</p>
          <Button variant="primary" onClick={() => setIsCreateModalOpen(true)}>
            <Plus className="w-4 h-4 mr-2" />
            {t('users.addUser')}
          </Button>
        </Card>
      )}

      {/* Create User Modal */}
      <Modal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        title={t('users.addUser')}
        size="lg"
      >
        <UserForm
          onSuccess={() => {
            setIsCreateModalOpen(false)
            refetch()
          }}
          onCancel={() => setIsCreateModalOpen(false)}
        />
      </Modal>

      {/* Edit User Modal */}
      <Modal
        isOpen={isEditModalOpen}
        onClose={() => {
          setIsEditModalOpen(false)
          setSelectedUser(null)
        }}
        title={t('users.editUser')}
        size="lg"
      >
        {selectedUser && (
          <UserForm
            user={selectedUser}
            onSuccess={() => {
              setIsEditModalOpen(false)
              setSelectedUser(null)
              refetch()
            }}
            onCancel={() => {
              setIsEditModalOpen(false)
              setSelectedUser(null)
            }}
          />
        )}
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={isDeleteModalOpen}
        onClose={() => {
          setIsDeleteModalOpen(false)
          setUserToDelete(null)
        }}
        title={t('users.deleteUser')}
      >
        <Alert variant="danger" className="mb-4">
          {t('users.deleteConfirm')}
        </Alert>
        {userToDelete && (
          <p className="text-secondary mb-4">
            {t('users.username')}: <strong>{userToDelete.username}</strong>
          </p>
        )}
        <div className="flex gap-3 justify-end">
          <Button
            variant="secondary"
            onClick={() => {
              setIsDeleteModalOpen(false)
              setUserToDelete(null)
            }}
          >
            {t('common.cancel')}
          </Button>
          <Button variant="danger" onClick={handleDelete}>
            {t('common.delete')}
          </Button>
        </div>
      </Modal>

      {/* View User Inbounds Modal */}
      <Modal
        isOpen={isInboundsModalOpen}
        onClose={() => {
          setIsInboundsModalOpen(false)
          setUserForInbounds(null)
        }}
        title={t('users.inboundsFor', { username: userForInbounds?.username || '' })}
        size="lg"
      >
        {userForInbounds && <UserInboundsView userId={userForInbounds.id} />}
      </Modal>

      {/* Subscription Links Modal */}
      {userForSubscription && (
        <SubscriptionLinks
          isOpen={isSubscriptionOpen}
          onClose={() => {
            setIsSubscriptionOpen(false)
            setUserForSubscription(null)
          }}
          user={userForSubscription}
        />
      )}
    </PageLayout>
  )
}

// Component to display user's inbounds
function UserInboundsView({ userId }: { userId: number }) {
  const { t } = useTranslation()
  const { data: inbounds, isLoading } = useUserInbounds(userId)

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Spinner size="lg" />
      </div>
    )
  }

  if (!inbounds || inbounds.length === 0) {
    return (
      <div className="text-center py-8">
        <p className="text-secondary">{t('users.noInboundsAssigned')}</p>
      </div>
    )
  }

  return (
    <div className="space-y-3">
      {inbounds.map((inbound: Inbound) => (
        <Card key={inbound.id} className="p-4">
          <div className="flex items-start justify-between">
            <div className="flex-1">
              <div className="flex items-center gap-2 mb-2">
                <h4 className="font-medium text-primary">{inbound.name}</h4>
                <Badge variant={inbound.is_enabled ? 'success' : 'default'}>
                  {inbound.is_enabled ? t('users.enabled') : t('users.disabled')}
                </Badge>
              </div>
              <div className="space-y-1 text-sm text-secondary">
                <div>
                  <span className="text-tertiary">{t('inbounds.protocol')}:</span> {inbound.protocol?.toUpperCase()}
                </div>
                <div>
                  <span className="text-tertiary">{t('inbounds.port')}:</span> {inbound.port}
                </div>
                {inbound.listen_address && (
                  <div>
                    <span className="text-tertiary">{t('inbounds.listenAddress')}:</span> {inbound.listen_address}
                  </div>
                )}
                {inbound.tls_enabled && (
                  <div>
                    <Badge variant="info" className="text-xs">TLS</Badge>
                  </div>
                )}
              </div>
            </div>
          </div>
        </Card>
      ))}
    </div>
  )
}
