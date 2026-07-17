import { useState } from 'react';
import { ArrowLeft, Info, Upload } from 'lucide-react';
import { api } from '@/api/client';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Switch } from '@/components/ui/switch';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { PushPhonePreview, SmsPhonePreview } from './PhonePreview';
import type { MsgType } from './shared';

/* Ports _CreateMsgDialog — a full-width create flow (rendered inside the tab
 * with a back arrow) for email / SMS / push, with live phone preview. */

function SectionLabel({ children }: { children: string }) {
  return (
    <div className="mb-3.5 text-[length:var(--text-caption)] font-semibold uppercase tracking-[0.6px] text-text-muted">
      {children}
    </div>
  );
}

function FieldLabel({ label, optional }: { label: string; optional?: boolean }) {
  return (
    <div className="mb-1.5 flex items-center gap-1">
      <span className="text-[length:var(--text-body)] text-text-secondary">{label}</span>
      {optional && (
        <span className="text-[length:var(--text-caption)] text-text-subtle">optional</span>
      )}
    </div>
  );
}

function Divider() {
  return <div className="my-5 border-t border-border" />;
}

function ScheduleField({
  value,
  onChange,
}: {
  value: string;
  onChange: (v: string) => void;
}) {
  return (
    <div>
      <FieldLabel label="Schedule" />
      <Select value={value} onValueChange={onChange}>
        <SelectTrigger className="w-full">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="Now">Now</SelectItem>
          <SelectItem value="Schedule">Schedule</SelectItem>
        </SelectContent>
      </Select>
      <div className="mt-2 flex items-center gap-1.5 text-[length:var(--text-label)] text-text-subtle">
        <Info size={12} />
        The message will be sent immediately
      </div>
    </div>
  );
}

export function CreateMessageView({
  type,
  onBack,
  onDone,
}: {
  type: MsgType;
  onBack: () => void;
  onDone: () => void;
}) {
  const [subject, setSubject] = useState('');
  const [message, setMessage] = useState('');
  const [to, setTo] = useState('');
  const [title, setTitle] = useState('');
  const [htmlMode, setHtmlMode] = useState(false);
  const [schedule, setSchedule] = useState('Now');
  const [sending, setSending] = useState(false);
  const [saving, setSaving] = useState(false);

  const heading =
    type === 'email'
      ? 'Create email message'
      : type === 'sms'
        ? 'Create SMS message'
        : 'Create push message';

  const send = async (draft: boolean) => {
    try {
      if (type === 'email') {
        const recipients = to
          .split(',')
          .map((e) => e.trim())
          .filter((e) => e.length > 0);
        await api.post('/messaging/messages/email', {
          to: recipients,
          subject,
          html: message,
          draft,
        });
      } else if (type === 'sms') {
        await api.post('/messaging/messages/sms', {
          to: to.trim(),
          body: message,
          draft,
        });
      } else {
        await api.post('/messaging/messages/push', {
          token: to.trim(),
          title,
          body: message,
          draft,
        });
      }
    } catch {
      // still close — backend returns 201 even if send fails asynchronously
    }
    onDone();
  };

  const create = async () => {
    setSending(true);
    try {
      await send(false);
    } finally {
      setSending(false);
    }
  };

  const saveDraft = async () => {
    setSaving(true);
    try {
      await send(true);
    } finally {
      setSaving(false);
    }
  };

  const busy = sending || saving;

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center gap-3 pb-1">
        <button
          type="button"
          onClick={onBack}
          className="text-text-muted transition-colors hover:text-text-primary"
          aria-label="Back"
        >
          <ArrowLeft size={20} />
        </button>
        <h2 className="text-[length:var(--text-h1)] font-semibold text-text-primary">
          {heading}
        </h2>
      </div>

      <div className="flex-1 overflow-y-auto pb-10 pt-6">
        {type === 'email' && (
          <div className="max-w-2xl">
            <SectionLabel>Message</SectionLabel>
            <FieldLabel label="Subject" />
            <Input
              value={subject}
              onChange={(e) => setSubject(e.target.value)}
              placeholder="Enter subject"
              autoFocus
            />
            <div className="mt-4">
              <FieldLabel label="Message" />
              <Textarea
                value={message}
                onChange={(e) => setMessage(e.target.value)}
                placeholder="Type here..."
                rows={5}
              />
            </div>
            <div className="mt-3 flex items-start gap-2.5">
              <Switch checked={htmlMode} onCheckedChange={setHtmlMode} />
              <div>
                <div className="text-[length:var(--text-body)] text-text-secondary">HTML mode</div>
                <div className="text-[length:var(--text-caption)] text-text-subtle">
                  Enable the HTML mode if your message contains HTML tags.
                </div>
              </div>
            </div>
            <Divider />
            <SectionLabel>Targets</SectionLabel>
            <FieldLabel label="To" />
            <Input
              value={to}
              onChange={(e) => setTo(e.target.value)}
              placeholder="user@example.com, other@example.com"
            />
            <Divider />
            <SectionLabel>Settings</SectionLabel>
            <ScheduleField value={schedule} onChange={setSchedule} />
          </div>
        )}

        {type === 'sms' && (
          <div className="flex items-start gap-8">
            <div className="max-w-xl flex-1">
              <SectionLabel>Message</SectionLabel>
              <FieldLabel label="Message" />
              <Textarea
                value={message}
                onChange={(e) => setMessage(e.target.value)}
                placeholder="Type here..."
                rows={5}
                maxLength={900}
                autoFocus
              />
              <div className="mt-1.5 text-right text-[length:var(--text-caption)] text-text-subtle">
                {message.length}/900
              </div>
              <Divider />
              <SectionLabel>Targets</SectionLabel>
              <FieldLabel label="To" />
              <Input
                value={to}
                onChange={(e) => setTo(e.target.value)}
                placeholder="+1234567890"
              />
              <Divider />
              <SectionLabel>Settings</SectionLabel>
              <ScheduleField value={schedule} onChange={setSchedule} />
            </div>
            <SmsPhonePreview message={message} />
          </div>
        )}

        {type === 'push' && (
          <div className="flex items-start gap-8">
            <div className="max-w-xl flex-1">
              <SectionLabel>Message</SectionLabel>
              <FieldLabel label="Title" />
              <Input
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="Enter title"
                autoFocus
              />
              <div className="mt-4">
                <FieldLabel label="Message" />
                <Textarea
                  value={message}
                  onChange={(e) => setMessage(e.target.value)}
                  placeholder="Type here..."
                  rows={4}
                  maxLength={1000}
                />
                <div className="mt-1.5 text-right text-[length:var(--text-caption)] text-text-subtle">
                  {message.length}/1000
                </div>
              </div>
              <div className="mt-4">
                <FieldLabel label="Media" optional />
                <div className="flex h-[90px] flex-col items-center justify-center rounded-[var(--radius)] border border-field-border bg-fill">
                  <Upload size={20} className="text-text-subtle" />
                  <div className="mt-1.5 text-[length:var(--text-body)] text-text-muted">
                    Select a file to upload
                  </div>
                  <div className="mt-1 flex items-center gap-2 text-[length:var(--text-caption)] text-text-subtle">
                    <span>Max file size: 1MB</span>
                    <span className="underline">Browse</span>
                  </div>
                </div>
              </div>
              <Divider />
              <SectionLabel>Targets</SectionLabel>
              <FieldLabel label="FCM Token" />
              <Input
                value={to}
                onChange={(e) => setTo(e.target.value)}
                placeholder="Enter device token"
              />
              <Divider />
              <SectionLabel>Settings</SectionLabel>
              <ScheduleField value={schedule} onChange={setSchedule} />
            </div>
            <PushPhonePreview title={title} message={message} />
          </div>
        )}
      </div>

      <div className="flex justify-end gap-2 border-t border-border py-4">
        <Button variant="ghost" onClick={onBack} disabled={busy}>
          Cancel
        </Button>
        <Button variant="outline" onClick={saveDraft} loading={saving} disabled={sending}>
          Save as draft
        </Button>
        <Button onClick={create} loading={sending} disabled={saving}>
          Create
        </Button>
      </div>
    </div>
  );
}
