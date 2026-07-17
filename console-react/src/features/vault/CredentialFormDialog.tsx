import { useEffect, useState } from 'react';
import { Eye, EyeOff } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { toast } from '@/components/toast';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Switch } from '@/components/ui/switch';
import { Checkbox } from '@/components/ui/checkbox';
import {
  FormDialog,
  FormField,
  SelectField,
  TextField,
} from '@/components/form-dialog';
import { CRED_TYPE_OPTIONS, type CredType } from './credentials';
import type { Row } from '@/components/data-table';

/* Create/edit a credential — ports _CredentialModal. On edit the secret value
 * must be re-entered to save (the API never returns it in the list). */
export function CredentialFormDialog({
  open,
  onOpenChange,
  existing,
  onSaved,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  existing: Row | null;
  onSaved: () => void;
}) {
  const isEdit = existing !== null;

  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [type, setType] = useState<CredType>('generic');
  const [secret, setSecret] = useState('');
  const [protectedFlag, setProtectedFlag] = useState(false);
  const [hasExpiry, setHasExpiry] = useState(false);
  const [expiresAt, setExpiresAt] = useState('');
  const [obscure, setObscure] = useState(true);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!open) return;
    setName(String(existing?.['name'] ?? ''));
    setDescription(String(existing?.['description'] ?? ''));
    setType((String(existing?.['type'] ?? 'generic') as CredType) || 'generic');
    setProtectedFlag(existing?.['protected'] === true);
    const exp = existing?.['expiresAt'] as string | undefined;
    setHasExpiry(!!exp);
    setExpiresAt(exp ? exp.slice(0, 10) : '');
    setSecret('');
    setObscure(true);
    setLoading(false);
  }, [open, existing]);

  const save = async () => {
    const trimmedName = name.trim();
    const trimmedSecret = secret.trim();
    if (!trimmedName) {
      toast.error('Name is required');
      return;
    }
    if (!trimmedSecret) {
      toast.error(isEdit ? 'Enter the secret value to update it' : 'Secret value is required');
      return;
    }
    setLoading(true);
    try {
      const body: Record<string, unknown> = {
        name: trimmedName,
        type,
        description: description.trim(),
        protected: protectedFlag,
        data: trimmedSecret,
        ...(hasExpiry && expiresAt
          ? { expiresAt: new Date(`${expiresAt}T00:00:00Z`).toISOString() }
          : {}),
      };
      if (isEdit) {
        await api.put(`/credentials/${String(existing?.['$id'])}`, body);
      } else {
        await api.post('/credentials', body);
      }
      toast.success(isEdit ? 'Credential updated' : 'Credential created');
      onOpenChange(false);
      onSaved();
    } catch (e) {
      toast.error(friendlyError(e));
    } finally {
      setLoading(false);
    }
  };

  return (
    <FormDialog
      open={open}
      onOpenChange={onOpenChange}
      title={isEdit ? 'Edit credential' : 'New credential'}
      submitLabel={isEdit ? 'Save' : 'Create'}
      loading={loading}
      submitDisabled={!name.trim() || !secret.trim()}
      onSubmit={save}
      width={480}
    >
      <TextField
        label="Name"
        placeholder="e.g. stripe-secret-key"
        value={name}
        onChange={(e) => setName(e.target.value)}
        autoFocus
      />
      <TextField
        label="Description (optional)"
        placeholder="What is this used for?"
        value={description}
        onChange={(e) => setDescription(e.target.value)}
      />
      <SelectField
        label="Type"
        value={type}
        onChange={setType}
        options={CRED_TYPE_OPTIONS}
      />

      <FormField
        label="Secret value"
        hint={isEdit ? '(re-enter the value to update it)' : undefined}
      >
        <div className="relative">
          {obscure ? (
            <Input
              type="password"
              className="pr-10 font-[family-name:var(--font-mono)]"
              placeholder={isEdit ? '(unchanged — enter to update)' : 'Paste the secret value'}
              value={secret}
              onChange={(e) => setSecret(e.target.value)}
            />
          ) : (
            <Textarea
              rows={4}
              className="pr-10 font-[family-name:var(--font-mono)]"
              placeholder={isEdit ? '(unchanged — enter to update)' : 'Paste the secret value'}
              value={secret}
              onChange={(e) => setSecret(e.target.value)}
            />
          )}
          <button
            type="button"
            onClick={() => setObscure((v) => !v)}
            className="absolute right-2 top-2 text-text-muted transition-colors hover:text-text-primary"
            aria-label={obscure ? 'Show secret' : 'Hide secret'}
          >
            {obscure ? <Eye size={16} /> : <EyeOff size={16} />}
          </button>
        </div>
      </FormField>

      <div className="flex items-start gap-3">
        <Switch checked={protectedFlag} onCheckedChange={setProtectedFlag} />
        <div className="flex flex-col">
          <span className="text-[length:var(--text-body)] text-text-primary">Protected</span>
          <span className="text-[length:var(--text-caption)] text-text-muted">
            Requires API key authentication to read. Client-side session tokens are blocked.
          </span>
        </div>
      </div>

      <div className="flex flex-col gap-2">
        <label className="flex items-center gap-2">
          <Checkbox
            checked={hasExpiry}
            onCheckedChange={(v) => {
              const on = v === true;
              setHasExpiry(on);
              if (!on) setExpiresAt('');
            }}
          />
          <span className="text-[length:var(--text-body)] text-text-primary">Set expiry date</span>
        </label>
        {hasExpiry && (
          <Input
            type="date"
            className="w-48"
            value={expiresAt}
            min={new Date().toISOString().slice(0, 10)}
            onChange={(e) => setExpiresAt(e.target.value)}
          />
        )}
      </div>
    </FormDialog>
  );
}
