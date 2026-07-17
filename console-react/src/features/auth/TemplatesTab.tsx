import { useState } from 'react';
import { Mail, Smartphone } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { FormDialog, TextField, TextAreaField } from '@/components/form-dialog';

interface Template {
  name: string;
  type: 'email' | 'sms';
  description: string;
}

const TEMPLATES: Template[] = [
  { name: 'Email verification', type: 'email', description: 'Sent when a user signs up to verify their email address.' },
  { name: 'Magic URL', type: 'email', description: "Passwordless sign-in link sent to the user's email." },
  { name: 'Password recovery', type: 'email', description: 'Sent when a user requests a password reset.' },
  { name: 'Invitation', type: 'email', description: 'Sent when a user is invited to join a team.' },
  { name: 'OTP verification', type: 'sms', description: 'SMS code sent for phone number verification.' },
];

export function TemplatesTab() {
  const [editing, setEditing] = useState<Template | null>(null);
  const [subject, setSubject] = useState('');
  const [body, setBody] = useState('');

  const openEditor = (t: Template) => {
    setSubject('');
    setBody('');
    setEditing(t);
  };

  return (
    <div className="pb-8">
      <h2 className="text-[length:var(--text-title)] font-semibold text-text-primary">Email &amp; SMS templates</h2>
      <p className="mt-1 text-[length:var(--text-body)] text-text-secondary">
        Customize the messages sent to your users during authentication flows.
      </p>

      <div className="mt-5 flex flex-col gap-2">
        {TEMPLATES.map((t) => {
          const Icon = t.type === 'email' ? Mail : Smartphone;
          return (
            <div key={t.name} className="flex items-center gap-3.5 rounded-[var(--radius)] border border-border bg-surface p-4">
              <div className="flex h-8 w-8 items-center justify-center rounded-[var(--radius-6)] bg-fill text-text-secondary">
                <Icon size={14} />
              </div>
              <div className="flex-1">
                <div className="text-[length:var(--text-control)] font-medium text-text-primary">{t.name}</div>
                <div className="mt-0.5 text-[length:var(--text-label)] text-text-subtle">{t.description}</div>
              </div>
              <Button variant="outline" size="sm" onClick={() => openEditor(t)}>
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
        onSubmit={() => setEditing(null)}
      >
        {editing?.type === 'email' && (
          <TextField label="Subject" placeholder="Subject line" value={subject} onChange={(e) => setSubject(e.target.value)} autoFocus />
        )}
        <TextAreaField
          label={editing?.type === 'sms' ? 'Message' : 'Body'}
          placeholder={editing?.type === 'sms' ? 'Your code is {{otp}}' : 'Message body — use {{name}}, {{url}} placeholders'}
          value={body}
          onChange={(e) => setBody(e.target.value)}
          rows={editing?.type === 'sms' ? 3 : 8}
        />
      </FormDialog>
    </div>
  );
}
