import { Bell, Mail, MessageSquare, Plus } from 'lucide-react';
import { api } from '@/api/client';
import { useResourceList } from '@/hooks/use-resource-list';
import { DataTable, type DataTableColumn, type Row } from '@/components/data-table';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { toMessageRow, type MessageRow } from './MessageDetail';
import { MessageStatusChip, typeIcon, typeName, type MsgType } from './shared';

const columns: DataTableColumn[] = [
  { key: 'id', label: 'Message ID', flex: 3 },
  { key: 'type', label: 'Type', flex: 2 },
  { key: 'status', label: 'Status', flex: 2 },
  { key: 'createdAt', label: 'Created', flex: 2 },
];

function CreateMenu({ onSelect }: { onSelect: (t: MsgType) => void }) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button size="sm">
          <Plus size={14} />
          Create message
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onClick={() => onSelect('email')}>
          <Mail size={14} />
          Email
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => onSelect('sms')}>
          <MessageSquare size={14} />
          SMS
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => onSelect('push')}>
          <Bell size={14} />
          Push notification
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function MessagesTab({
  projectId,
  onCreate,
  onSelect,
}: {
  projectId: string | undefined;
  onCreate: (type: MsgType) => void;
  onSelect: (msg: MessageRow) => void;
}) {
  const list = useResourceList({
    endpoint: '/messaging/messages',
    itemsKey: 'messages',
    scope: [projectId],
  });

  return (
    <DataTable
      columns={columns}
      rows={list.rows}
      getCellValue={(row, key) => {
        if (key === 'type') return typeName(String(row.type ?? ''));
        if (key === 'id') return String(row.id ?? row.$id ?? '');
        return String(row[key] ?? '');
      }}
      cellRender={(row, key) =>
        key === 'status' ? <MessageStatusChip status={String(row.status ?? '')} /> : undefined
      }
      rowIcon={(row) => typeIcon(String(row.type ?? ''))}
      onRowClick={(row: Row) => onSelect(toMessageRow(row))}
      onDeleteRow={async (row: Row) => {
        const id = String(row.id ?? row.$id ?? '');
        await api.delete(`/messaging/messages/${id}`);
        list.refetch();
      }}
      deleteTitle="Delete message?"
      deleteMessage="This cannot be undone. The message and its delivery record will be removed."
      createWidget={<CreateMenu onSelect={onCreate} />}
      searchHint="Search by type, status, or ID"
      searchValue={list.search}
      onSearchChange={list.setSearch}
      onSearch={list.runSearch}
      total={list.total}
      perPage={list.perPage}
      page={list.page}
      onPerPageChange={list.setPerPage}
      onPrev={list.prevPage}
      onNext={list.nextPage}
      itemLabel="messages"
      filters={[
        { key: 'type', label: 'Type', options: [
          { value: 'email', label: 'Email' },
          { value: 'sms', label: 'SMS' },
          { value: 'push', label: 'Push' },
        ] },
        { key: 'status', label: 'Status', options: [
          { value: 'processing', label: 'Processing' },
          { value: 'sent', label: 'Sent' },
          { value: 'failed', label: 'Failed' },
          { value: 'draft', label: 'Draft' },
        ] },
      ]}
      filterValues={list.filters}
      onFiltersChange={list.setFilters}
      emptyIcon={MessageSquare}
      emptyTitle="No messages"
      emptySubtitle="Create a message to send email, SMS, or push notifications."
      loading={list.isLoading}
      error={list.error}
      onRetry={list.refetch}
    />
  );
}
