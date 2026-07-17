import { useMemo, useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Info } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Switch } from '@/components/ui/switch';
import { Button } from '@/components/ui/button';
import { StatusChip } from '@/components/status-chip';
import { RichTextEditor } from '@/components/rich-text-editor';
import { toast } from '@/components/toast';
import {
  contentKeys,
  dateOnly,
  parseFields,
  slugify,
  type ContentRow,
  type FieldDef,
} from './shared';

type FieldValue = string | boolean;

function initialValues(fields: FieldDef[], entry: ContentRow | null): Record<string, FieldValue> {
  const data = (entry?.data as Record<string, unknown>) ?? {};
  const out: Record<string, FieldValue> = {};
  for (const f of fields) {
    if (f.type === 'boolean') out[f.key] = data[f.key] === true;
    else out[f.key] = data[f.key] != null ? String(data[f.key]) : '';
  }
  return out;
}

export function EntryEditor({
  projectId,
  type,
  entry,
  onBack,
  onSaved,
}: {
  projectId?: string;
  type: ContentRow;
  entry: ContentRow | null;
  onBack: () => void;
  onSaved: () => void;
}) {
  const qc = useQueryClient();
  const typeId = String(type.$id);
  const isNew = entry === null;
  const fields = useMemo(() => parseFields(type), [type]);
  const localized = type.localization === true;

  const [locale, setLocale] = useState(String(entry?.locale ?? 'en'));
  const [values, setValues] = useState<Record<string, FieldValue>>(() =>
    initialValues(fields, entry),
  );
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const status = String(entry?.status ?? 'draft');
  const typeName = String(type.name ?? 'Entry');
  const entryLabel = isNew ? 'New entry' : String(entry?.slug ?? entry?.$id ?? 'Entry');

  const setValue = (key: string, v: FieldValue) => setValues((s) => ({ ...s, [key]: v }));

  const deriveSlug = (): string => {
    const first = fields.find((f) => ['text', 'richtext', 'slug'].includes(f.type));
    if (!first) return '';
    return slugify(String(values[first.key] ?? ''));
  };

  const collectData = (): Record<string, unknown> => {
    const out: Record<string, unknown> = {};
    for (const f of fields) {
      const v = values[f.key];
      if (f.type === 'boolean') out[f.key] = v === true;
      else if (f.type === 'number') out[f.key] = Number(v || 0) || 0;
      else out[f.key] = v ?? '';
    }
    return out;
  };

  const save = async (publish: boolean) => {
    setSaving(true);
    setError(null);
    try {
      const data = collectData();
      if (isNew) {
        const res = await api.post(`/content/types/${typeId}/entries`, {
          slug: deriveSlug(),
          locale: locale.trim() || 'en',
          data,
        });
        if (publish) {
          const newId = String((res.data as ContentRow)?.$id ?? '');
          if (newId) await api.patch(`/content/types/${typeId}/entries/${newId}/publish`);
        }
      } else {
        const entryId = String(entry!.$id);
        await api.put(`/content/types/${typeId}/entries/${entryId}`, { data });
        if (publish) await api.patch(`/content/types/${typeId}/entries/${entryId}/publish`);
      }
      qc.invalidateQueries({ queryKey: ['content-entries', projectId, typeId] });
      qc.invalidateQueries({ queryKey: contentKeys.types(projectId) });
      toast.success(publish ? 'Entry published' : 'Draft saved');
      onSaved();
    } catch (e) {
      setError(friendlyError(e));
      setSaving(false);
    }
  };

  return (
    <div className="flex h-full flex-col gap-6 p-6 md:p-8">
      {/* Integrated header */}
      <div className="flex items-center gap-2.5">
        <button
          type="button"
          onClick={onBack}
          className="text-text-secondary transition-colors hover:text-text-primary"
          aria-label="Back"
        >
          <ArrowLeft size={16} />
        </button>
        <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">
          {typeName} · {entryLabel}
        </h1>
        <div className="ml-auto flex items-center gap-2">
          {error && (
            <span className="text-[length:var(--text-label)] text-status-danger">{error}</span>
          )}
          <Button variant="outline" loading={saving} onClick={() => save(false)}>
            Save draft
          </Button>
          <Button
            loading={saving}
            onClick={() => save(true)}
            className="bg-[#10B981] text-white hover:bg-[#0EA271]"
          >
            {status === 'published' ? 'Update & publish' : 'Publish'}
          </Button>
        </div>
      </div>

      {/* Body: form + sidebar */}
      <div className="flex flex-1 gap-7 overflow-hidden">
        <div className="flex-1 overflow-y-auto pr-1">
          <div className="flex max-w-3xl flex-col gap-5">
            {localized && (
              <Field label="Locale">
                <Input value={locale} onChange={(e) => setLocale(e.target.value)} placeholder="en" />
              </Field>
            )}
            {fields.map((f) => (
              <Field key={f.key} label={f.label || f.key} required={f.required}>
                <FieldInput
                  field={f}
                  value={values[f.key]}
                  onChange={(v) => setValue(f.key, v)}
                />
              </Field>
            ))}
          </div>
        </div>

        <div className="w-52 shrink-0 overflow-y-auto">
          {isNew ? (
            <div className="rounded-[var(--radius)] border border-border bg-surface p-3">
              <div className="flex items-center gap-1.5 text-[length:var(--text-label)] font-semibold text-text-secondary">
                <Info size={13} className="text-text-subtle" />
                Draft
              </div>
              <p className="mt-1.5 text-[length:var(--text-label)] leading-relaxed text-text-muted">
                Saved as a draft. Use Publish to make it live.
              </p>
            </div>
          ) : (
            <div className="flex flex-col gap-4">
              <SidebarSection title="Status">
                <StatusChip label={status} />
              </SidebarSection>
              <SidebarSection title="Version">
                <span className="text-[length:var(--text-body)] text-text-secondary">
                  v{String(entry?.version ?? 1)}
                </span>
              </SidebarSection>
              <VersionHistory projectId={projectId} typeId={typeId} entryId={String(entry!.$id)} />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function FieldInput({
  field,
  value,
  onChange,
}: {
  field: FieldDef;
  value: string | boolean | undefined;
  onChange: (v: string | boolean) => void;
}) {
  switch (field.type) {
    case 'boolean': {
      const on = value === true;
      return (
        <div className="flex items-center gap-2">
          <Switch checked={on} onCheckedChange={onChange} />
          <span className="text-[length:var(--text-body)] text-text-secondary">
            {on ? 'True' : 'False'}
          </span>
        </div>
      );
    }
    case 'richtext':
      return (
        <RichTextEditor
          value={typeof value === 'string' ? value : ''}
          onChange={onChange}
          placeholder={'Write in Markdown…\n\nTip: **bold**, *italic*, `code`, ## headings'}
          minRows={14}
        />
      );
    case 'number':
      return (
        <Input
          type="number"
          value={typeof value === 'string' ? value : ''}
          onChange={(e) => onChange(e.target.value)}
          placeholder="0"
        />
      );
    case 'date':
      return (
        <Input
          value={typeof value === 'string' ? value : ''}
          onChange={(e) => onChange(e.target.value)}
          placeholder="YYYY-MM-DD"
        />
      );
    case 'seo':
      return (
        <div className="rounded-[var(--radius-6)] border border-border bg-fill p-3">
          <p className="mb-2 text-[length:var(--text-caption)] text-text-muted">
            SEO meta (stored as JSON in this field)
          </p>
          <Textarea
            value={typeof value === 'string' ? value : ''}
            onChange={(e) => onChange(e.target.value)}
            placeholder='{"title":"","description":"","image":""}'
            rows={3}
            className="font-[family-name:var(--font-mono)]"
          />
        </div>
      );
    default:
      return (
        <Input
          value={typeof value === 'string' ? value : ''}
          onChange={(e) => onChange(e.target.value)}
          placeholder=""
        />
      );
  }
}

function Field({
  label,
  required,
  children,
}: {
  label: string;
  required?: boolean;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-[length:var(--text-label)] font-medium text-text-secondary">
        {label}
        {required && <span className="ml-0.5 text-status-danger">*</span>}
      </span>
      {children}
    </div>
  );
}

function SidebarSection({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-[length:var(--text-caption)] font-semibold uppercase tracking-wide text-text-muted">
        {title}
      </span>
      {children}
    </div>
  );
}

function VersionHistory({
  projectId,
  typeId,
  entryId,
}: {
  projectId?: string;
  typeId: string;
  entryId: string;
}) {
  const versionsQuery = useQuery({
    queryKey: contentKeys.versions(projectId, typeId, entryId),
    queryFn: async () => {
      const res = await api.get(`/content/types/${typeId}/entries/${entryId}/versions`);
      return ((res.data as ContentRow)?.versions as ContentRow[]) ?? [];
    },
  });

  const versions = (versionsQuery.data ?? []).slice(0, 5);

  return (
    <SidebarSection title="Version history">
      {versionsQuery.isLoading ? (
        <span className="text-[length:var(--text-caption)] text-text-muted">Loading…</span>
      ) : versions.length === 0 ? (
        <span className="text-[length:var(--text-caption)] text-text-muted">No versions</span>
      ) : (
        <div className="flex flex-col gap-1.5">
          {versions.map((v, i) => (
            <div key={i} className="flex items-center justify-between">
              <span className="font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] text-text-secondary">
                v{String(v.version ?? '')}
              </span>
              <span className="text-[length:var(--text-caption)] text-text-muted">
                {dateOnly(v.$createdAt ?? v.createdAt)}
              </span>
            </div>
          ))}
        </div>
      )}
    </SidebarSection>
  );
}
