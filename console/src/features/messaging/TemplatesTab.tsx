import { useState } from 'react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { Bell, FileText, Mail, MessageSquare, Plus, Trash2 } from 'lucide-react';
import { api } from '@/api/client';
import { Button } from '@/components/ui/button';
import {
  ConfirmDialog,
  FormDialog,
  SelectField,
  TextAreaField,
  TextField,
} from '@/components/form-dialog';
import { ErrorState } from '@/components/error-state';
import type { MsgType } from './shared';

interface Template extends Record<string, unknown> {
  name?: string;
  type?: string;
  subject?: string;
}

const TYPE_COLOR: Record<string, string> = {
  email: '#3472A4',
  sms: '#10B981',
  push: '#F59E0B',
};

function typeIconFor(type: string) {
  if (type === 'email') return Mail;
  if (type === 'sms') return MessageSquare;
  return Bell;
}

export function TemplatesTab({ projectId }: { projectId: string | undefined }) {
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState('');
  const [type, setType] = useState<MsgType>('email');
  const [subject, setSubject] = useState('');
  const [body, setBody] = useState('');
  const [pendingDelete, setPendingDelete] = useState<string | null>(null);

  const query = useQuery({
    queryKey: ['/messaging/templates', projectId],
    queryFn: async () => {
      const res = await api.get('/messaging/templates');
      return (res.data as { templates?: Template[] }).templates ?? [];
    },
  });

  const resetForm = () => {
    setName('');
    setType('email');
    setSubject('');
    setBody('');
  };

  const create = useMutation({
    mutationFn: async () => {
      await api.post('/messaging/templates', {
        templateId: 'unique()',
        name: name.trim(),
        type,
        subject: subject.trim(),
        body: body.trim(),
        variables: [],
      });
    },
    onSuccess: () => {
      setCreating(false);
      resetForm();
      void query.refetch();
    },
  });

  const del = useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/messaging/templates/${id}`);
    },
    onSuccess: () => {
      setPendingDelete(null);
      void query.refetch();
    },
  });

  const templates = query.data ?? [];

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center">
        <span className="text-[length:var(--text-body)] text-text-secondary">
          {templates.length} template{templates.length === 1 ? '' : 's'}
        </span>
        <span className="flex-1" />
        <Button size="sm" onClick={() => setCreating(true)}>
          <Plus size={14} />
          New Template
        </Button>
      </div>

      {query.error ? (
        <ErrorState error={query.error} onRetry={() => void query.refetch()} />
      ) : query.isLoading ? (
        <div className="py-16 text-center text-[length:var(--text-body)] text-text-muted">
          Loading…
        </div>
      ) : templates.length === 0 ? (
        <div className="flex flex-col items-center justify-center px-6 py-16 text-center">
          <FileText size={40} className="text-text-subtle" />
          <div className="mt-3 text-[length:var(--text-subhead)] font-medium text-text-primary">
            No templates yet
          </div>
          <div className="mt-1 text-[length:var(--text-body)] text-text-secondary">
            Create reusable message templates with variables.
          </div>
        </div>
      ) : (
        <div className="overflow-hidden rounded-[var(--radius-10)] border border-border">
          {templates.map((t, i) => {
            const id = String(t.$id ?? '');
            const tType = String(t.type ?? 'email');
            const color = TYPE_COLOR[tType] ?? '#6B7280';
            const Icon = typeIconFor(tType);
            return (
              <div
                key={id || i}
                className="group flex items-center gap-3 border-b border-border px-4 py-3 last:border-0"
              >
                <div
                  className="flex h-9 w-9 items-center justify-center rounded-[var(--radius)]"
                  style={{ backgroundColor: `color-mix(in srgb, ${color} 12%, transparent)` }}
                >
                  <Icon size={16} style={{ color }} />
                </div>
                <div className="flex min-w-0 flex-1 flex-col">
                  <span className="truncate text-[length:var(--text-body)] font-medium text-text-primary">
                    {String(t.name ?? '')}
                  </span>
                  {t.subject && (
                    <span className="truncate text-[length:var(--text-label)] text-text-subtle">
                      {String(t.subject)}
                    </span>
                  )}
                </div>
                <span
                  className="rounded-full px-2 py-[3px] text-[length:var(--text-2xs)] font-semibold"
                  style={{
                    color,
                    backgroundColor: `color-mix(in srgb, ${color} 12%, transparent)`,
                  }}
                >
                  {tType.toUpperCase()}
                </span>
                <button
                  type="button"
                  onClick={() => setPendingDelete(id)}
                  className="rounded-[var(--radius-6)] p-1.5 text-text-subtle opacity-0 transition-all hover:bg-fill hover:text-[var(--color-danger)] group-hover:opacity-100"
                  aria-label="Delete template"
                >
                  <Trash2 size={14} />
                </button>
              </div>
            );
          })}
        </div>
      )}

      <FormDialog
        open={creating}
        onOpenChange={(o) => {
          setCreating(o);
          if (!o) resetForm();
        }}
        title="New Template"
        subtitle="Create a reusable message with {{variable}} placeholders"
        width={500}
        submitLabel="Create"
        loading={create.isPending}
        submitDisabled={!name.trim()}
        onSubmit={() => create.mutate()}
      >
        <TextField
          label="Name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="Welcome Email"
          autoFocus
        />
        <SelectField
          label="Type"
          value={type}
          onChange={setType}
          options={[
            { value: 'email', label: 'Email' },
            { value: 'sms', label: 'SMS' },
            { value: 'push', label: 'Push' },
          ]}
        />
        <TextField
          label="Subject"
          value={subject}
          onChange={(e) => setSubject(e.target.value)}
          placeholder="Hello {{name}}!"
        />
        <TextAreaField
          label="Body"
          value={body}
          onChange={(e) => setBody(e.target.value)}
          placeholder="Hi {{name}}, welcome to {{appName}}!"
          rows={5}
          className="font-[family-name:var(--font-mono)]"
        />
      </FormDialog>

      <ConfirmDialog
        open={pendingDelete !== null}
        onOpenChange={(o) => !o && setPendingDelete(null)}
        title="Delete template"
        message="Are you sure? This action cannot be undone."
        loading={del.isPending}
        onConfirm={() => pendingDelete && del.mutate(pendingDelete)}
      />
    </div>
  );
}
