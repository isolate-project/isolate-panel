import { useState } from 'preact/hooks'

import { PageLayout } from '../components/layout/PageLayout'
import { PageHeader } from '../components/layout/PageHeader'
import { Card, CardContent } from '../components/ui/Card'
import { Button } from '../components/ui/Button'
import { Badge } from '../components/ui/Badge'
import { Skeleton } from '../components/ui/Skeleton'
import { Modal } from '../components/ui/Modal'
import { Alert } from '../components/ui/Alert'
import { Input } from '../components/ui/Input'
import { Select } from '../components/ui/Select'
import { Drawer } from '../components/ui/Drawer'
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator } from '../components/ui/DropdownMenu'
import { useInbounds, useDeleteInbound, useAssignUser, useUnassignUser } from '../hooks/useInbounds'
import { useUsers } from '../hooks/useUsers'
import { useQuery } from '../hooks/useQuery'
import { inboundApi } from '../api/endpoints'
import { InboundForm } from '../components/forms/InboundForm'
import type { Inbound, User } from '../types'
import { Plus, Edit, Trash2, Users as UsersIcon, Search, MoreVertical, Globe, ShieldAlert, Cpu } from 'lucide-preact'
import { useTranslation } from 'react-i18next'


interface ActionMenuProps {
  inbound: Inbound;
  openDropdownId: number | null;
  setOpenDropdownId: (id: number | null) => void;
  onManageUsers: (inbound: Inbound) => void;
  onEdit: (inbound: Inbound) => void;
  onDelete: (inbound: Inbound) => void;
}

const InboundActionMenu = ({ 
  inbound, 
  openDropdownId, 
  setOpenDropdownId, 
  onManageUsers, 
  onEdit, 
  onDelete 
}: ActionMenuProps) => {
  const { t } = useTranslation()
  return (
    <DropdownMenu>
      <DropdownMenuTrigger onClick={() => setOpenDropdownId(openDropdownId === inbound.id ? null : inbound.id)}>
        <Button variant="ghost" size="icon" className="h-8 w-8 text-text-tertiary">
          <MoreVertical className="h-4 w-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent isOpen={openDropdownId === inbound.id} onClose={() => setOpenDropdownId(null)}>
        <DropdownMenuItem onClick={() => { setOpenDropdownId(null); onManageUsers(inbound); }}>
          <UsersIcon className="mr-2 h-4 w-4 text-text-secondary" /> {t('inbounds.manageUsers')}
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => { 
          setOpenDropdownId(null); 
          onEdit(inbound);
        }}>
          <Edit className="mr-2 h-4 w-4 text-text-secondary" /> {t('inbounds.editInbound')}
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={() => { setOpenDropdownId(null); onDelete(inbound); }} variant="danger">
          <Trash2 className="mr-2 h-4 w-4" /> {t('common.delete')}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

export function Inbounds() {
  const { t } = useTranslation()
  const { data: inbounds, isLoading, refetch } = useInbounds()
  const { mutate: deleteInbound } = useDeleteInbound()
  const { data: usersResponse } = useUsers()
  
  const [searchTerm, setSearchTerm] = useState('')
  const [protocolFilter, setProtocolFilter] = useState<string>('all')
  const [openDropdownId, setOpenDropdownId] = useState<number | null>(null)
  
  // Modals & Drawers state
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false)
  const [inboundToDelete, setInboundToDelete] = useState<Inbound | null>(null)
  const [isUsersModalOpen, setIsUsersModalOpen] = useState(false)
  const [inboundForUsers, setInboundForUsers] = useState<Inbound | null>(null)
  
  // Drawer state for Create/Edit
  const [isDrawerOpen, setIsDrawerOpen] = useState(false)
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create')
  const [inboundToEdit, setInboundToEdit] = useState<Inbound | null>(null)

  const { mutate: assignUser, isLoading: isAssigning } = useAssignUser()
  const { mutate: unassignUser, isLoading: isUnassigning } = useUnassignUser()

  const allInbounds: Inbound[] = Array.isArray(inbounds) ? inbounds : []
  const allUsers: User[] = Array.isArray(usersResponse?.users) ? usersResponse.users : (Array.isArray(usersResponse) ? usersResponse as unknown as User[] : [])

  const { data: assignedUsersData, refetch: refetchAssigned } = useQuery<{ users: User[]; total: number }>(
    inboundForUsers ? `inbound-users-${inboundForUsers.id}` : 'inbound-users-none',
    () => inboundApi.getUsers(inboundForUsers!.id).then((res) => res.data as { users: User[]; total: number }),
    { enabled: !!inboundForUsers }
  )
  const assignedUserIds = new Set((assignedUsersData?.users ?? []).map((u: User) => u.id))

  const filteredInbounds = allInbounds.filter((inbound) => {
    const matchesSearch = searchTerm
      ? inbound.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
        inbound.protocol.toLowerCase().includes(searchTerm.toLowerCase())
      : true
    const matchesProtocol = protocolFilter === 'all' || inbound.protocol === protocolFilter
    return matchesSearch && matchesProtocol
  })

  const handleDelete = async () => {
    if (inboundToDelete) {
      await deleteInbound(inboundToDelete.id)
      setIsDeleteModalOpen(false)
      setInboundToDelete(null)
      refetch()
    }
  }

  return (
    <PageLayout>
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-6">
        <PageHeader
          title={t('inbounds.title')}
          description={t('inbounds.description')}
        />
        <Button onClick={() => { setDrawerMode('create'); setInboundToEdit(null); setIsDrawerOpen(true); }}>
          <Plus className="w-4 h-4 mr-2" />
          {t('inbounds.addInbound')}
        </Button>
      </div>

      {/* Control Bar */}
      <Card className="mb-6 rounded-2xl shadow-sm border-white/5">
        <CardContent className="p-4 sm:p-2 sm:px-4">
          <div className="flex flex-col sm:flex-row gap-4 sm:items-center">
            <div className="flex-1 relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-text-tertiary" />
              <Input
                type="text"
                placeholder={t('inbounds.searchPlaceholder')}
                value={searchTerm}
                onChange={(e) => setSearchTerm((e.target as HTMLInputElement).value)}
                className="pl-10 h-10 bg-transparent border-none focus:ring-0 shadow-none text-base sm:text-sm placeholder:text-text-tertiary"
              />
            </div>
            <div className="h-px sm:h-8 w-full sm:w-px bg-border-primary mx-2 hidden sm:block"></div>
            <div className="w-full sm:w-48">
              <Select
                value={protocolFilter}
                onChange={(e) => setProtocolFilter((e.target as HTMLSelectElement).value)}
                options={[
                  { value: 'all', label: t('inbounds.allProtocols') },
                  { value: 'vless', label: 'VLESS' },
                  { value: 'vmess', label: 'VMess' },
                  { value: 'trojan', label: 'Trojan' },
                  { value: 'shadowsocks', label: 'Shadowsocks' },
                ]}
              />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Content Grid */}
      {isLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {Array(6).fill(0).map((_, i) => <Skeleton key={i} className="h-48 w-full rounded-2xl" />)}
        </div>
      ) : filteredInbounds.length > 0 ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {filteredInbounds.map((inbound) => (
            <Card key={inbound.id} className="relative overflow-hidden group hover:shadow-lg transition-all duration-300">
              <CardContent className="p-0">
                {/* Header Section */}
                <div className="p-5 border-b border-border-primary/50 bg-bg-secondary/30 flex justify-between items-start">
                  <div>
                    <div className="flex items-center gap-2 mb-1.5">
                      <div className="w-8 h-8 rounded-lg bg-color-primary/10 text-color-primary flex items-center justify-center border border-color-primary/20">
                        <Globe className="w-4 h-4" />
                      </div>
                      <h3 className="font-semibold text-text-primary text-lg">{inbound.name}</h3>
                    </div>
                    <Badge variant={inbound.is_enabled ? 'success' : 'secondary'} showDot className="uppercase tracking-wider text-[10px]">
                      {inbound.is_enabled ? t('inbounds.active') : t('inbounds.disabled')}
                    </Badge>
                  </div>
                  <InboundActionMenu 
                    inbound={inbound} 
                    openDropdownId={openDropdownId}
                    setOpenDropdownId={setOpenDropdownId}
                    onManageUsers={(inb) => { setInboundForUsers(inb); setIsUsersModalOpen(true); }}
                    onEdit={(inb) => { setInboundToEdit(inb); setDrawerMode('edit'); setIsDrawerOpen(true); }}
                    onDelete={(inb) => { setInboundToDelete(inb); setIsDeleteModalOpen(true); }}
                  />
                </div>

                {/* Details Section */}
                <div className="p-5 space-y-4">
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-1">
                      <p className="text-xs text-text-tertiary uppercase tracking-wider">{t('inbounds.protocol')}</p>
                      <p className="text-sm font-medium text-text-primary flex items-center gap-1.5">
                        {inbound.protocol.toUpperCase()}
                      </p>
                    </div>
                    <div className="space-y-1">
                      <p className="text-xs text-text-tertiary uppercase tracking-wider">{t('inbounds.port')}</p>
                      <p className="text-sm font-medium text-text-primary font-mono bg-bg-tertiary px-1.5 py-0.5 rounded inline-block">
                        {inbound.port}
                      </p>
                    </div>
                  </div>

                  <div className="space-y-1">
                    <p className="text-xs text-text-tertiary uppercase tracking-wider">{t('inbounds.core')}</p>
                    <p className="text-sm font-medium text-text-secondary flex items-center gap-2">
                       <Cpu className="w-3.5 h-3.5" />
                       {inbound.core?.name || t('inbounds.noCoreAssigned')}
                    </p>
                  </div>

                  {/* Capabilities Tags */}
                  <div className="pt-2 flex flex-wrap gap-1.5">
                    {inbound.tls_enabled && <Badge variant="glass" className="text-[10px] bg-indigo-500/10 text-indigo-500 border-indigo-500/20">TLS</Badge>}
                    {inbound.reality_enabled && <Badge variant="glass" className="text-[10px] bg-purple-500/10 text-purple-500 border-purple-500/20">REALITY</Badge>}
                    <Badge variant="glass" className="text-[10px] text-text-secondary">TCP</Badge>
                  </div>
                </div>

                {/* Footer Action */}
                <div className="p-3 bg-bg-tertiary/30 border-t border-border-primary/50">
                  <Button variant="ghost" fullWidth className="text-sm text-text-secondary hover:text-color-primary" onClick={() => { setInboundForUsers(inbound); setIsUsersModalOpen(true); }}>
                    <UsersIcon className="w-4 h-4 mr-2" />
                    {t('inbounds.manageAccess')}
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      ) : (
        <Card className="text-center py-16">
          <CardContent className="flex flex-col items-center">
            <div className="w-16 h-16 bg-bg-tertiary rounded-full flex items-center justify-center mb-4">
              <ShieldAlert className="w-8 h-8 text-text-tertiary" />
            </div>
            <p className="text-lg font-medium text-text-primary">{t('inbounds.noMatchingInbounds')}</p>
            <p className="text-text-secondary mb-6 mt-1">{t('inbounds.createHint')}</p>
            <Button onClick={() => { setDrawerMode('create'); setInboundToEdit(null); setIsDrawerOpen(true); }}><Plus className="w-4 h-4 mr-2" /> {t('inbounds.addInbound')}</Button>
          </CardContent>
        </Card>
      )}

      {/* Drawer: Create / Edit Inbound */}
      <Drawer
        isOpen={isDrawerOpen}
        onClose={() => setIsDrawerOpen(false)}
        title={drawerMode === 'create' ? t('inbounds.addInbound') : t('inbounds.editInbound')}
        description={drawerMode === 'create' ? t('inbounds.createDescription') : t('inbounds.editDescription')}
        size="lg"
      >
        <InboundForm 
          inbound={inboundToEdit}
          onSuccess={() => { setIsDrawerOpen(false); refetch() }} 
          onCancel={() => setIsDrawerOpen(false)} 
        />
      </Drawer>

      <Modal isOpen={isDeleteModalOpen} onClose={() => { setIsDeleteModalOpen(false); setInboundToDelete(null) }} title={t('inbounds.deleteInbound')}>
        <Alert variant="danger" className="mb-4">{t('inbounds.deleteConfirm')}</Alert>
        {inboundToDelete && <p className="text-text-secondary mb-6">{t('inbounds.deleteConfirmation')} <strong>{inboundToDelete.name}</strong>?</p>}
        <div className="flex gap-3 justify-end">
          <Button variant="outline" onClick={() => { setIsDeleteModalOpen(false); setInboundToDelete(null) }}>{t('common.cancel')}</Button>
          <Button variant="destructive" onClick={handleDelete}>{t('common.delete')}</Button>
        </div>
      </Modal>

      <Modal isOpen={isUsersModalOpen} onClose={() => { setIsUsersModalOpen(false); setInboundForUsers(null) }} title={`${t('inbounds.manageAccess')} - ${inboundForUsers?.name || ''}`} size="lg">
        {inboundForUsers && (
          <div className="space-y-4">
            <p className="text-sm text-text-secondary mb-4">Select which users are allowed to connect through this inbound.</p>
            <div className="space-y-2 max-h-[60vh] overflow-y-auto pr-2">
              {allUsers.map((user) => {
                const isAssigned = assignedUserIds.has(user.id)
                const isBusy = isAssigning || isUnassigning
                return (
                  <div key={user.id} className="flex items-center justify-between p-3 border border-border-primary rounded-xl hover:bg-bg-tertiary/30 transition-colors">
                    <div className="flex items-center gap-3">
                      <div className="w-8 h-8 rounded-full bg-color-primary/10 text-color-primary flex items-center justify-center font-bold text-sm">
                        {user.username.charAt(0).toUpperCase()}
                      </div>
                      <div>
                        <div className="font-medium text-text-primary text-sm">{user.username}</div>
                        <Badge variant={user.is_active ? 'success' : 'secondary'} className="text-[10px] mt-0.5">{user.is_active ? 'Active' : 'Disabled'}</Badge>
                      </div>
                    </div>
                    <div>
                      {isAssigned ? (
                        <Button
                          variant="outline"
                          size="sm"
                          disabled={isBusy}
                          onClick={() => unassignUser({ inboundId: inboundForUsers.id, userId: user.id }).then(() => refetchAssigned()).catch(() => {})}
                        >
                          Unassign
                        </Button>
                      ) : (
                        <Button
                          variant="outline"
                          size="sm"
                          disabled={isBusy}
                          onClick={() => assignUser({ inboundId: inboundForUsers.id, userId: user.id }).then(() => refetchAssigned()).catch(() => {})}
                        >
                          Assign
                        </Button>
                      )}
                    </div>
                  </div>
                )
              })}
            </div>
          </div>
        )}
      </Modal>
    </PageLayout>
  )
}
