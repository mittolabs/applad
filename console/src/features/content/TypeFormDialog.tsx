import { useEffect, useState } from 'react';
import { Asterisk, Minus, Plus, X } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { FormDialog, TextField } from '@/components/form-dialog';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { cn } from '@/lib/utils';
import {
  FIELD_TYPES,
  parseFields,
  slugify,
  type ContentRow,
  type FieldDef,
} from './shared';

/* Ports _TypeFormDialog + _FieldRow — create/edit a content type with a
 * dynamic list of custom fields, plus versioning/localization toggles. */
export function TypeFormDialog({
  open,
  onOpenChange,
  type,
  onSaved,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  type: ContentRow | null;
  onSaved: () => void;
}) {
  const isEdit = type !== null;
  const [name, setName] = useState('');
  const [slug, setSlug] = useState('');
  const [versioning, setVersioning] = useState(false);
  const [localization, setLocalization] = useState(false);
  const [fields, setFields] = useState<FieldDef[]>([]);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) return;
    setName(String(type?.name ?? ''));
    setSlug(String(type?.slug ?? ''));
    setVersioning(type?.versioning === true);
    setLocalization(type?.localization === true);
    setFields(parseFields(type));
    setError(null);
  }, [open, type]);

  const addField = () =>
    setFields((f) => [...f, { key: '', label: '', type: 'text', required: false }]);
  const removeField = (i: number) => setFields((f) => f.filter((_, idx) => idx !== i));
  const updateField = (i: number, patch: Partial<FieldDef>) =>
    setFields((f) => f.map((fd, idx) => (idx === i ? { ...fd, ...patch } : fd)));

  const submit = async () => {
    setSaving(true);
    setError(null);
    try {
      if (isEdit) {
        await api.put(`/content/types/${String(type!.$id)}`, {
          name: name.trim(),
          fields,
        });
      } else {
        await api.post('/content/types', {
          name: name.trim(),
          slug: slug.trim() || slugify(name),
          fields,
          versioning,
          localization,
        });
      }
      onOpenChange(false);
      onSaved();
    } catch (e) {
      setError(friendlyError(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <FormDialog
      open={open}
      onOpenChange={onOpenChange}
      title={isEdit ? `Edit ${String(type?.name ?? 'type')}` : 'New content type'}
      submitLabel="Save"
      loading={saving}
      submitDisabled={!name.trim()}
      width={560}
      onSubmit={submit}
    >
      <TextField
        label="Name"
        value={name}
        onChange={(e) => setName(e.target.value)}
        placeholder="Blog posts"
        autoFocus
      />
      <TextField
        label="Slug (API identifier)"
        value={slug}
        onChange={(e) => setSlug(e.target.value)}
        placeholder="blog-posts"
        hint="Auto-generated from name if left blank"
        disabled={isEdit}
      />

      <div className="flex items-center gap-8">
        <label className="flex items-center gap-2">
          <Switch checked={versioning} onCheckedChange={setVersioning} disabled={isEdit} />
          <span className="text-[length:var(--text-body)] text-text-secondary">Versioning</span>
        </label>
        <label className="flex items-center gap-2">
          <Switch checked={localization} onCheckedChange={setLocalization} disabled={isEdit} />
          <span className="text-[length:var(--text-body)] text-text-secondary">Localization</span>
        </label>
      </div>

      <div className="flex flex-col gap-2.5">
        <div className="flex items-center justify-between">
          <Label>Fields</Label>
          <button
            type="button"
            onClick={addField}
            className="inline-flex items-center gap-1 text-[length:var(--text-label)] font-medium text-[var(--color-accent)]"
          >
            <Plus size={13} />
            Add field
          </button>
        </div>
        {fields.length === 0 ? (
          <p className="text-[length:var(--text-label)] text-text-muted">
            No fields yet. Add a field to define the schema.
          </p>
        ) : (
          fields.map((f, i) => (
            <div
              key={i}
              className="flex items-center gap-2 rounded-[var(--radius-6)] border border-border bg-fill p-2.5"
            >
              <Input
                value={f.key}
                onChange={(e) => updateField(i, { key: e.target.value })}
                placeholder="key"
                className="h-8 flex-1"
              />
              <Input
                value={f.label}
                onChange={(e) => updateField(i, { label: e.target.value })}
                placeholder="Label"
                className="h-8 flex-1"
              />
              <Select value={f.type} onValueChange={(v) => updateField(i, { type: v })}>
                <SelectTrigger className="h-8 w-28 shrink-0">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {FIELD_TYPES.map((t) => (
                    <SelectItem key={t} value={t}>
                      {t}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <button
                type="button"
                title="Required"
                onClick={() => updateField(i, { required: !f.required })}
                className={cn(
                  'shrink-0 transition-colors',
                  f.required ? 'text-[var(--color-accent)]' : 'text-text-muted hover:text-text-secondary',
                )}
              >
                {f.required ? <Asterisk size={14} /> : <Minus size={14} />}
              </button>
              <button
                type="button"
                onClick={() => removeField(i)}
                className="shrink-0 text-text-muted transition-colors hover:text-status-danger"
                aria-label="Remove field"
              >
                <X size={14} />
              </button>
            </div>
          ))
        )}
      </div>

      {error && (
        <p className="text-[length:var(--text-caption)] text-status-danger">{error}</p>
      )}
    </FormDialog>
  );
}
