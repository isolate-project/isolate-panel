import { useState, useMemo } from 'preact/hooks'
import { PageLayout } from '../components/layout/PageLayout'
import { PageHeader } from '../components/layout/PageHeader'
import { Card, CardContent } from '../components/ui/Card'
import { Button } from '../components/ui/Button'
import { Badge } from '../components/ui/Badge'
import { Progress } from '../components/ui/Progress'
import { Skeleton } from '../components/ui/Skeleton'
import { Modal } from '../components/ui/Modal'
import { Alert } from '../components/ui/Alert'
import { Input } from '../components/ui/Input'
import { Select } from '../components/ui/Select'
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator } from '../components/ui/DropdownMenu'
import { useUsers, useDeleteUser, useRegenerateCredentials, useUserInbounds } from '../hooks/useUsers'
import { UserForm } from '../components/forms/UserForm'
import { SubscriptionLinks } from '../components/features/SubscriptionLinks'
import type { User, Inbound } from '../types'
import { Plus, Edit, Trash2, RefreshCw, Copy, List, Search, Link as LinkIcon, MoreVertical, CalendarDays, ShieldAlert } from 'lucide-preact'
import { useTranslation } from 'react-i18next'

export function Users() {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const [limit, setLimit] = useState(20)
  const [searchTerm, setSearchTerm] = useState('')
  const [statusFilter, setStatusFilter] = useState<'all' | 'active' | 'inactive'>('all')
  const [openDropdownId, setOpenDropdownId] = useState<number | null>(null)
  
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

  const users = response?.users || []
  const total = response?.total || 0
  const totalPages = Math.ceil(total / limit)

  const filteredUsers = useMemo(() => {
    let filtered = users
    if (searchTerm) {
      const term = searchTerm.toLowerCase()
      filtered = filtered.filter((user: User) =>
        user.username?.toLowerCase().includes(term) ||
        user.email?.toLowerCase().includes(term) ||
        user.uuid?.toLowerCase().includes(term)
      )
    }
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

interface UserActionMenuProps {
  user: User;
  openDropdownId: number | null;
  setOpenDropdownId: (id: number | null) => void;
  onCopyToken: (token: string) => void;
  onViewSubscription: (user: User) => void;
  onViewInbounds: (user: User) => void;
  onEdit: (user: User) => void;
  onRegenerate: (userId: number) => void;
  onDelete: (user: User) => void;
  t: (key: string) => string;
}

const UserActionMenu = ({
  user,
  openDropdownId,
  setOpenDropdownId,
  onCopyToken,
  onViewSubscription,
  onViewInbounds,
  onEdit,
  onRegenerate,
  onDelete,
  t
}: UserActionMenuProps) => (
  <DropdownMenu>
    <DropdownMenuTrigger onClick={() => setOpenDropdownId(openDropdownId === user.id ? null : user.id)}>
      <Button variant="ghost" size="icon" className="h-8 w-8 text-text-tertiary">
        <MoreVertical className="h-4 w-4" />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent isOpen={openDropdownId === user.id} onClose={() => setOpenDropdownId(null)}>
      <DropdownMenuItem onClick={() => { setOpenDropdownId(null); onCopyToken(user.subscription_token) }}>
        <Copy className="mr-2 h-4 w-4 text-text-secondary" /> {t('users.copyToken')}
      </DropdownMenuItem>
      <DropdownMenuItem onClick={() => { setOpenDropdownId(null); onViewSubscription(user); }}>
        <LinkIcon className="mr-2 h-4 w-4 text-text-secondary" /> {t('subscriptions.title')}
      </DropdownMenuItem>
      <DropdownMenuSeparator />
      <DropdownMenuItem onClick={() => { setOpenDropdownId(null); onViewInbounds(user) }}>
        <List className="mr-2 h-4 w-4 text-text-secondary" /> {t('users.viewInbounds')}
      </DropdownMenuItem>
      <DropdownMenuItem onClick={() => { setOpenDropdownId(null); onEdit(user); }}>
        <Edit className="mr-2 h-4 w-4 text-text-secondary" /> {t('users.editUser')}
      </DropdownMenuItem>
      <DropdownMenuItem onClick={() => { setOpenDropdownId(null); onRegenerate(user.id) }}>
        <RefreshCw className="mr-2 h-4 w-4 text-text-secondary" /> {t('users.regenerateCredentials')}
      </DropdownMenuItem>
      <DropdownMenuSeparator />
      <DropdownMenuItem onClick={() => { setOpenDropdownId(null); onDelete(user); }} variant="danger">
        <Trash2 className="mr-2 h-4 w-4" /> {t('users.deleteUser')}
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
)

const TrafficDisplay = ({ used, total, formatBytes }: { used: number, total: number | null, formatBytes: (b: number) => string }) => {
  if (!total) return <span className="text-sm font-medium text-text-primary">{formatBytes(used)} <span className="text-text-tertiary text-xs font-normal ml-1">Total Limit: ∞</span></span>
  const percent = Math.min((used / total) * 100, 100)
  return (
    <div className="w-full max-w-[200px]">
      <div className="flex justify-between items-center text-xs mb-1.5">
        <span className="font-medium text-text-primary">{formatBytes(used)}</span>
        <span className="text-text-tertiary">{formatBytes(total)}</span>
      </div>
      <Progress 
        value={percent} 
        indicatorClassName={percent > 90 ? 'bg-color-danger' : percent > 75 ? 'bg-color-warning' : 'bg-color-success'}
      />
    </div>
  )
}

  return (
    <PageLayout>
      <PageHeader
        title={t('users.title')}
        description={t('users.description')}
        actions={
          <Button onClick={() => setIsCreateModalOpen(true)}>
            <Plus className="w-4 h-4 mr-2" />
            {t('users.addUser')}
          </Button>
        }
      />

      <Card className="mb-6 rounded-2xl shadow-sm border-white/5">
        <CardContent className="p-4 sm:p-2 sm:px-4">
          <div className="flex flex-col sm:flex-row gap-4 sm:items-center">
            <div className="flex-1 relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-text-tertiary" />
              <Input
                type="text"
                placeholder={t('users.searchPlaceholder') || 'Search users...'}
                value={searchTerm}
                onChange={(e: any) => setSearchTerm(e.target.value)}
                className="pl-10 h-10 bg-transparent border-none focus:ring-0 shadow-none text-base sm:text-sm placeholder:text-text-tertiary"
              />
            </div>
            <div className="h-px sm:h-8 w-full sm:w-px bg-border-primary mx-2 hidden sm:block"></div>
            <div className="flex gap-4 sm:gap-2">
              <Select
                value={statusFilter}
                onChange={(e: any) => setStatusFilter(e.target.value)}
                options={[
                  { value: 'all', label: t('common.all') || 'All Status' },
                  { value: 'active', label: t('common.active') || 'Active' },
                  { value: 'inactive', label: t('common.inactive') || 'Inactive' },
                ]}
              />
              <Select
                value={limit.toString()}
                onChange={(e: any) => { setLimit(Number(e.target.value)); setPage(1) }}
                options={[
                  { value: '10', label: '10 / page' }, { value: '20', label: '20 / page' }, { value: '50', label: '50 / page' }
                ]}
              />
            </div>
          </div>
        </CardContent>
      </Card>

      {isLoading ? (
        <Card><CardContent className="p-6 space-y-4">{Array(5).fill(0).map((_, i) => <Skeleton key={i} className="h-16 w-full rounded-xl" />)}</CardContent></Card>
      ) : filteredUsers && filteredUsers.length > 0 ? (
        <>
          <Card className="hidden md:block overflow-hidden shadow-sm">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm whitespace-nowrap">
                <thead className="bg-bg-tertiary/50 text-text-secondary border-b border-border-primary uppercase text-xs tracking-wider">
                  <tr>
                    <th className="px-6 py-4 font-semibold">User details</th>
                    <th className="px-6 py-4 font-semibold">Status</th>
                    <th className="px-6 py-4 font-semibold w-[250px]">Traffic</th>
                    <th className="px-6 py-4 font-semibold">Created</th>
                    <th className="px-6 py-4 text-right"></th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border-primary/50 text-text-primary">
                  {filteredUsers.map((user: User) => (
                    <tr key={user.id} className="hover:bg-bg-hover/50 transition-colors group">
                      <td className="px-6 py-3">
                        <div className="flex items-center gap-3">
                          <div className="w-10 h-10 rounded-full bg-color-primary/10 text-color-primary flex items-center justify-center font-bold text-lg leading-none shrink-0">
                            {user.username.charAt(0).toUpperCase()}
                          </div>
                          <div>
                            <p className="font-semibold text-text-primary">{user.username}</p>
                            <p className="text-xs text-text-tertiary">ID: {user.uuid?.substring(0, 8)}</p>
                          </div>
                        </div>
                      </td>
                      <td className="px-6 py-3">
                        <Badge variant={user.is_active ? 'success' : 'secondary'} showDot className="uppercase tracking-wider text-[10px]">
                          {user.is_active ? 'Active' : 'Disabled'}
                        </Badge>
                      </td>
                      <td className="px-6 py-3">
                        <TrafficDisplay used={user.traffic_used_bytes || 0} total={user.traffic_limit_bytes} formatBytes={formatBytes} />
                      </td>
                      <td className="px-6 py-3 text-text-secondary">
                        <div className="flex items-center gap-2 text-sm">
                          <CalendarDays className="w-4 h-4 text-text-tertiary" />
                          {new Date(user.created_at).toLocaleDateString()}
                        </div>
                      </td>
                      <td className="px-6 py-3 text-right">
                        <UserActionMenu 
                          user={user} 
                          openDropdownId={openDropdownId}
                          setOpenDropdownId={setOpenDropdownId}
                          onCopyToken={copyToClipboard}
                          onViewSubscription={(u) => { setUserForSubscription(u); setIsSubscriptionOpen(true); }}
                          onViewInbounds={handleViewInbounds}
                          onEdit={(u) => { setSelectedUser(u); setIsEditModalOpen(true); }}
                          onRegenerate={handleRegenerate}
                          onDelete={(u) => { setUserToDelete(u); setIsDeleteModalOpen(true); }}
                          t={t}
                        />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </Card>

          <div className="grid grid-cols-1 gap-4 md:hidden">
            {filteredUsers.map((user: User) => (
              <Card key={user.id} className="relative overflow-hidden">
                <CardContent className="p-4 space-y-4">
                  <div className="flex justify-between items-start">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 rounded-full bg-color-primary/10 text-color-primary flex items-center justify-center font-bold text-lg leading-none">
                        {user.username.charAt(0).toUpperCase()}
                      </div>
                      <div>
                        <p className="font-semibold text-base text-text-primary">{user.username}</p>
                        <Badge variant={user.is_active ? 'success' : 'secondary'} showDot className="mt-1 lowercase text-[10px]">
                          {user.is_active ? 'active' : 'disabled'}
                        </Badge>
                      </div>
                    </div>
                    <div>
                      <UserActionMenu 
                        user={user} 
                        openDropdownId={openDropdownId}
                        setOpenDropdownId={setOpenDropdownId}
                        onCopyToken={copyToClipboard}
                        onViewSubscription={(u) => { setUserForSubscription(u); setIsSubscriptionOpen(true); }}
                        onViewInbounds={handleViewInbounds}
                        onEdit={(u) => { setSelectedUser(u); setIsEditModalOpen(true); }}
                        onRegenerate={handleRegenerate}
                        onDelete={(u) => { setUserToDelete(u); setIsDeleteModalOpen(true); }}
                        t={t}
                      />
                    </div>
                  </div>
                  
                  <div className="bg-bg-tertiary/30 rounded-xl p-3 border border-border-primary/50">
                    <TrafficDisplay used={user.traffic_used_bytes || 0} total={user.traffic_limit_bytes} formatBytes={formatBytes} />
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>

          <div className="mt-6 flex flex-col sm:flex-row items-center justify-between gap-4">
            <p className="text-sm text-text-secondary">
              Showing <span className="font-medium text-text-primary">{((page - 1) * limit) + 1}</span> to <span className="font-medium text-text-primary">{Math.min(page * limit, total)}</span> of <span className="font-medium text-text-primary">{total}</span> users
            </p>
            <div className="flex gap-2">
              <Button variant="outline" size="sm" onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page === 1}>Previous</Button>
              <Button variant="outline" size="sm" onClick={() => setPage(p => Math.min(totalPages, p + 1))} disabled={page === totalPages}>Next</Button>
            </div>
          </div>
        </>
      ) : (
        <Card className="text-center py-16">
          <CardContent className="flex flex-col items-center">
            <div className="w-16 h-16 bg-bg-tertiary rounded-full flex items-center justify-center mb-4">
              <ShieldAlert className="w-8 h-8 text-text-tertiary" />
            </div>
            <p className="text-lg font-medium text-text-primary">No users found</p>
            <p className="text-text-secondary mb-6 mt-1">Create a user to give them access to the internet.</p>
            <Button onClick={() => setIsCreateModalOpen(true)}><Plus className="w-4 h-4 mr-2" /> Add User</Button>
          </CardContent>
        </Card>
      )}

      <Modal isOpen={isCreateModalOpen} onClose={() => setIsCreateModalOpen(false)} title={t('users.addUser')} size="lg">
        <UserForm onSuccess={() => { setIsCreateModalOpen(false); refetch() }} onCancel={() => setIsCreateModalOpen(false)} />
      </Modal>

      <Modal isOpen={isEditModalOpen} onClose={() => { setIsEditModalOpen(false); setSelectedUser(null) }} title={t('users.editUser')} size="lg">
        {selectedUser && <UserForm user={selectedUser} onSuccess={() => { setIsEditModalOpen(false); setSelectedUser(null); refetch() }} onCancel={() => { setIsEditModalOpen(false); setSelectedUser(null) }} />}
      </Modal>

      <Modal isOpen={isDeleteModalOpen} onClose={() => { setIsDeleteModalOpen(false); setUserToDelete(null) }} title={t('users.deleteUser')}>
        <Alert variant="danger" className="mb-4">This action cannot be undone.</Alert>
        {userToDelete && <p className="text-text-secondary mb-6">Are you sure you want to permanently delete user <strong>{userToDelete.username}</strong>?</p>}
        <div className="flex gap-3 justify-end">
          <Button variant="outline" onClick={() => { setIsDeleteModalOpen(false); setUserToDelete(null) }}>{t('common.cancel')}</Button>
          <Button variant="destructive" onClick={handleDelete}>{t('common.delete')}</Button>
        </div>
      </Modal>

      <Modal isOpen={isInboundsModalOpen} onClose={() => { setIsInboundsModalOpen(false); setUserForInbounds(null) }} title={`Inbounds for ${userForInbounds?.username || ''}`} size="lg">
        {userForInbounds && <UserInboundsView userId={userForInbounds.id} />}
      </Modal>

      {userForSubscription && (
        <SubscriptionLinks isOpen={isSubscriptionOpen} onClose={() => { setIsSubscriptionOpen(false); setUserForSubscription(null) }} user={userForSubscription} />
      )}
    </PageLayout>
  )
}

function UserInboundsView({ userId }: { userId: number }) {
  const { t } = useTranslation()
  const { data: inbounds, isLoading } = useUserInbounds(userId)

  if (isLoading) return <div className="flex items-center justify-center py-8"><Skeleton className="h-12 w-full" /></div>
  if (!inbounds || inbounds.length === 0) return <div className="text-center py-8 text-text-secondary">{t('users.noInboundsAssigned')}</div>

  return (
    <div className="space-y-3">
      {inbounds.map((inbound: Inbound) => (
        <Card key={inbound.id}>
          <CardContent className="p-4 flex justify-between items-center">
            <div>
              <div className="flex items-center gap-2 mb-1">
                <h4 className="font-medium text-text-primary text-base">{inbound.name}</h4>
                <Badge variant={inbound.is_enabled ? 'success' : 'secondary'} className="text-[10px] uppercase">
                  {inbound.is_enabled ? t('users.enabled') : t('users.disabled')}
                </Badge>
              </div>
              <p className="text-xs text-text-secondary font-mono bg-bg-tertiary px-2 py-1 rounded inline-block">
                {inbound.protocol}://{inbound.listen_address || '*'}:{inbound.port}
              </p>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
