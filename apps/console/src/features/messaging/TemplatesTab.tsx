import { useState } from 'react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { Bell, FileText, Mail, MessageSquare, Pencil, Plus, Send, Trash2 } from 'lucide-react';
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
  body?: string;
  variables?: string[];
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
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editingVariables, setEditingVariables] = useState<string[]>([]);
  const [name, setName] = useState('');
  const [type, setType] = useState<MsgType>('email');
  const [subject, setSubject] = useState('');
  const [body, setBody] = useState('');
  const [pendingDelete, setPendingDelete] = useState<string | null>(null);

  const [sending, setSending] = useState<Template | null>(null);
  const [sendTo, setSendTo] = useState('');
  const [sendVars, setSendVars] = useState<Record<string, string>>({});

  const query = useQuery({
    queryKey: ['/messaging/templates', projectId],
    queryFn: async () => {
      const res = await api.get('/messaging/templates');
      return (res.data as { templates?: Template[] }).templates ?? [];
    },
  });

  const resetForm = () => {
    setEditingId(null);
    setEditingVariables([]);
    setName('');
    setType('email');
    setSubject('');
    setBody('');
  };

  const openCreate = () => {
    resetForm();
    setDialogOpen(true);
  };

  const openEdit = (t: Template) => {
    setEditingId(String(t.$id ?? ''));
    setEditingVariables(Array.isArray(t.variables) ? t.variables : []);
    setName(String(t.name ?? ''));
    setType((String(t.type ?? 'email') as MsgType) || 'email');
    setSubject(String(t.subject ?? ''));
    setBody(String(t.body ?? ''));
    setDialogOpen(true);
  };

  const save = useMutation({
    mutationFn: async () => {
      const payload = {
        name: name.trim(),
        type,
        subject: subject.trim(),
        body: body.trim(),
        variables: editingId ? editingVariables : [],
      };
      if (editingId) {
        await api.put(`/messaging/templates/${editingId}`, payload);
      } else {
        await api.post('/messaging/templates', { templateId: 'unique()', ...payload });
      }
    },
    onSuccess: () => {
      setDialogOpen(false);
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

  const openSend = (t: Template) => {
    setSending(t);
    setSendTo('');
    setSendVars(
      Object.fromEntries((Array.isArray(t.variables) ? t.variables : []).map((v) => [v, ''])),
    );
  };

  const send = useMutation({
    mutationFn: async () => {
      const id = String(sending?.$id ?? '');
      const to = sendTo
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean);
      await api.post(`/messaging/templates/${id}/send`, { to, variables: sendVars });
    },
    onSuccess: () => {
      setSending(null);
      setSendTo('');
      setSendVars({});
    },
  });

  const templates = query.data ?? [];
  const sendRecipients = sendTo
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center">
        <span className="text-[length:var(--text-body)] text-text-secondary">
          {templates.length} template{templates.length === 1 ? '' : 's'}
        </span>
        <span className="flex-1" />
        <Button size="sm" onClick={openCreate}>
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
                  onClick={() => openSend(t)}
                  className="rounded-[var(--radius-6)] p-1.5 text-text-subtle opacity-0 transition-all hover:bg-fill hover:text-[var(--color-accent)] group-hover:opacity-100"
                  aria-label="Send template"
                >
                  <Send size={14} />
                </button>
                <button
                  type="button"
                  onClick={() => openEdit(t)}
                  className="rounded-[var(--radius-6)] p-1.5 text-text-subtle opacity-0 transition-all hover:bg-fill hover:text-text-primary group-hover:opacity-100"
                  aria-label="Edit template"
                >
                  <Pencil size={14} />
                </button>
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
        open={dialogOpen}
        onOpenChange={(o) => {
          setDialogOpen(o);
          if (!o) resetForm();
        }}
        title={editingId ? 'Edit Template' : 'New Template'}
        subtitle="Create a reusable message with {{variable}} placeholders"
        width={500}
        submitLabel={editingId ? 'Save' : 'Create'}
        loading={save.isPending}
        submitDisabled={!name.trim()}
        onSubmit={() => save.mutate()}
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

      <FormDialog
        open={sending !== null}
        onOpenChange={(o) => {
          if (!o) {
            setSending(null);
            setSendTo('');
            setSendVars({});
          }
        }}
        title="Send Template"
        subtitle={sending ? `Send "${String(sending.name ?? '')}" to recipients` : undefined}
        width={500}
        submitLabel="Send"
        loading={send.isPending}
        submitDisabled={sendRecipients.length === 0}
        onSubmit={() => send.mutate()}
      >
        <TextField
          label="Recipients"
          value={sendTo}
          onChange={(e) => setSendTo(e.target.value)}
          placeholder="alice@example.com, bob@example.com"
          hint="Comma-separated list of recipients"
          autoFocus
        />
        {(Array.isArray(sending?.variables) ? sending!.variables! : []).map((v) => (
          <TextField
            key={v}
            label={v}
            value={sendVars[v] ?? ''}
            onChange={(e) => setSendVars((prev) => ({ ...prev, [v]: e.target.value }))}
            placeholder={`Value for {{${v}}}`}
          />
        ))}
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
