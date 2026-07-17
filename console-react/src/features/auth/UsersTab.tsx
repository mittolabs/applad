import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { User, Users } from 'lucide-react';
import { api } from '@/api/client';
import { useResourceList } from '@/hooks/use-resource-list';
import { DataTable, type DataTableColumn, type Row } from '@/components/data-table';
import { StatusChip } from '@/components/status-chip';
import { FormDialog, TextField } from '@/components/form-dialog';
import { relativeTime } from './format';

const COLUMNS: DataTableColumn[] = [
  { key: '$id', label: 'User ID', flex: 3 },
  { key: 'name', label: 'Name', flex: 3, sortable: true },
  { key: 'email', label: 'Email', flex: 4, sortable: true },
  { key: 'status', label: 'Status', flex: 2 },
  { key: '$createdAt', label: 'Joined', flex: 2, sortable: true },
];

export function UsersTab({ projectId }: { projectId: string }) {
  const list = useResourceList({ endpoint: '/users', itemsKey: 'users', scope: [projectId] });
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');

  const createUser = useMutation({
    mutationFn: () =>
      api.post('/users', { userId: 'unique()', email, password, name }).then((r) => r.data),
    onSuccess: () => {
      setCreating(false);
      setName('');
      setEmail('');
      setPassword('');
      list.refetch();
    },
  });

  return (
    <>
      <DataTable
        columns={COLUMNS}
        rows={list.rows as Row[]}
        getCellValue={(row, key) => {
          switch (key) {
            case '$id':
              return String(row['$id'] ?? '');
            case 'name':
              return String(row['name'] ?? '') || 'Anonymous';
            case 'email':
              return String(row['email'] ?? '');
            case 'status':
              return row['status'] ? 'Active' : 'Disabled';
            case '$createdAt':
              return relativeTime(String(row['$createdAt'] ?? ''));
            default:
              return '';
          }
        }}
        cellRender={(row, key) => {
          if (key !== 'status') return undefined;
          const active = Boolean(row['status']);
          if (!active) return <StatusChip label="disabled" />;
          const verified = Boolean(row['emailVerification']);
          return <StatusChip label={verified ? 'verified' : 'unverified'} />;
        }}
        rowIcon={() => User}
        onDeleteRow={async (row) => {
          await api.delete(`/users/${row['$id']}`);
          list.refetch();
        }}
        deleteTitle="Delete user"
        deleteMessage="Are you sure you want to delete this user? This action cannot be undone."
        createLabel="Create user"
        onCreate={() => setCreating(true)}
        searchHint="Search by name, email, or ID"
        searchValue={list.search}
        onSearchChange={list.setSearch}
        onSearch={list.runSearch}
        total={list.total}
        perPage={list.perPage}
        page={list.page}
        onPerPageChange={list.setPerPage}
        onPrev={list.prevPage}
        onNext={list.nextPage}
        itemLabel="users"
        emptyIcon={Users}
        emptyTitle="No users yet"
        emptySubtitle="Add users manually or let them sign up through your app."
        loading={list.isLoading}
        error={list.error}
        onRetry={list.refetch}
      />

      <FormDialog
        open={creating}
        onOpenChange={setCreating}
        title="Create user"
        subtitle="Add a new user to your project"
        submitLabel="Create"
        loading={createUser.isPending}
        submitDisabled={!email.trim() || !password.trim()}
        onSubmit={() => createUser.mutate()}
      >
        <TextField label="Name" placeholder="Full name" value={name} onChange={(e) => setName(e.target.value)} autoFocus />
        <TextField label="Email" placeholder="user@example.com" value={email} onChange={(e) => setEmail(e.target.value)} />
        <TextField
          label="Password"
          type="password"
          placeholder="At least 8 characters"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
      </FormDialog>
    </>
  );
}
