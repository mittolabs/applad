import { useEffect, useState } from 'react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { Mail, Smartphone } from 'lucide-react';
import { api } from '@/api/client';
import { Button } from '@/components/ui/button';
import { FormDialog, TextField, TextAreaField } from '@/components/form-dialog';
import { ErrorState } from '@/components/error-state';

interface TemplateMeta {
  /** Server key under auth_config.emailTemplates. */
  key: string;
  name: string;
  type: 'email' | 'sms';
  description: string;
  /** Documented {{variables}} the sender substitutes for this flow. */
  variables: string[];
}

/* Only the flows that actually pass through the auth mailer / SMS sender are
 * editable. Team invitations are surfaced as a one-time link rather than
 * emailed, so there is no template to customize and no row here. */
const TEMPLATES: TemplateMeta[] = [
  {
    key: 'verification',
    name: 'Email verification',
    type: 'email',
    description: 'Sent when a user signs up to verify their email address.',
    variables: ['url', 'name', 'email'],
  },
  {
    key: 'magic',
    name: 'Magic URL',
    type: 'email',
    description: "Passwordless sign-in link sent to the user's email.",
    variables: ['url', 'email'],
  },
  {
    key: 'recovery',
    name: 'Password recovery',
    type: 'email',
    description: 'Sent when a user requests a password reset.',
    variables: ['url', 'email'],
  },
  {
    key: 'otp',
    name: 'OTP verification',
    type: 'sms',
    description: 'SMS code sent for phone number verification.',
    variables: ['otp'],
  },
];

interface StoredTemplate {
  subject?: string;
  body?: string;
}

type TemplateMap = Record<string, StoredTemplate>;

export function TemplatesTab({ projectId }: { projectId: string | undefined }) {
  const [editing, setEditing] = useState<TemplateMeta | null>(null);
  const [subject, setSubject] = useState('');
  const [body, setBody] = useState('');

  const query = useQuery({
    queryKey: ['auth-templates', projectId],
    enabled: !!projectId,
    queryFn: async () => {
      const res = await api.get(`/projects/${projectId}/auth/templates`);
      return (res.data as { templates?: TemplateMap }).templates ?? {};
    },
  });

  const stored = query.data ?? {};

  // Load the saved copy into the editor whenever a template is opened or the
  // fetched data arrives.
  useEffect(() => {
    if (!editing) return;
    const cur = stored[editing.key] ?? {};
    setSubject(cur.subject ?? '');
    setBody(cur.body ?? '');
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [editing]);

  const save = useMutation({
    mutationFn: async () => {
      if (!editing) return;
      // Merge the edited template into the full map and PUT the whole set. An
      // empty body clears the customization (server drops it, reverting to the
      // built-in copy).
      const next: TemplateMap = { ...stored };
      if (body.trim() === '' && subject.trim() === '') {
        delete next[editing.key];
      } else {
        next[editing.key] = { subject: subject.trim(), body: body.trim() };
      }
      await api.put(`/projects/${projectId}/auth/templates`, { templates: next });
    },
    onSuccess: () => {
      setEditing(null);
      void query.refetch();
    },
  });

  if (query.isError) {
    return <ErrorState error={query.error} onRetry={() => query.refetch()} />;
  }

  const isCustomized = (key: string) => {
    const t = stored[key];
    return !!(t && (t.body || t.subject));
  };

  return (
    <div className="pb-8">
      <h2 className="text-[length:var(--text-title)] font-semibold text-text-primary">Email &amp; SMS templates</h2>
      <p className="mt-1 text-[length:var(--text-body)] text-text-secondary">
        Customize the messages sent to your users during authentication flows. Leave a template blank to use
        the built-in copy. Email delivery requires SMTP to be configured; OTP requires an SMS provider.
      </p>

      <div className="mt-5 flex flex-col gap-2">
        {TEMPLATES.map((t) => {
          const Icon = t.type === 'email' ? Mail : Smartphone;
          return (
            <div key={t.key} className="flex items-center gap-3.5 rounded-[var(--radius)] border border-border bg-surface p-4">
              <div className="flex h-8 w-8 items-center justify-center rounded-[var(--radius-6)] bg-fill text-text-secondary">
                <Icon size={14} />
              </div>
              <div className="flex-1">
                <div className="flex items-center gap-2">
                  <span className="text-[length:var(--text-control)] font-medium text-text-primary">{t.name}</span>
                  {isCustomized(t.key) && (
                    <span className="rounded-[var(--radius-6)] bg-fill px-1.5 py-0.5 text-[length:var(--text-label)] text-text-secondary">
                      Customized
                    </span>
                  )}
                </div>
                <div className="mt-0.5 text-[length:var(--text-label)] text-text-subtle">{t.description}</div>
              </div>
              <Button variant="outline" size="sm" disabled={query.isLoading} onClick={() => setEditing(t)}>
                Edit
              </Button>
            </div>
          );
        })}
      </div>

      <FormDialog
        open={editing !== null}
        onOpenChange={(o) => !o && setEditing(null)}
        title={editing ? `Edit ${editing.name}` : ''}
        subtitle={editing?.type === 'sms' ? 'SMS message template' : 'Email message template'}
        submitLabel="Save"
        width={560}
        loading={save.isPending}
        onSubmit={() => save.mutate()}
      >
        {editing?.type === 'email' && (
          <TextField
            label="Subject"
            placeholder="Leave blank to keep the default subject"
            value={subject}
            onChange={(e) => setSubject(e.target.value)}
            autoFocus
          />
        )}
        <TextAreaField
          label={editing?.type === 'sms' ? 'Message' : 'Body'}
          placeholder={
            editing?.type === 'sms'
              ? 'Your code is {{otp}}'
              : 'Leave blank to use the built-in copy'
          }
          value={body}
          onChange={(e) => setBody(e.target.value)}
          rows={editing?.type === 'sms' ? 3 : 8}
        />
        {editing && (
          <p className="text-[length:var(--text-label)] text-text-subtle">
            Variables: {editing.variables.map((v) => `{{${v}}}`).join(', ')}
          </p>
        )}
        {save.isError && (
          <p className="text-[length:var(--text-label)] text-[color:var(--status-danger)]">Could not save. Please try again.</p>
        )}
      </FormDialog>
    </div>
  );
}
