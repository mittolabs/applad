import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  ArrowLeft,
  ArrowRight,
  BarChart3,
  ChevronRight,
  GitBranch,
  Key,
  Pencil,
  PieChart,
  Plus,
  Trash2,
  User,
  UserCog,
  Users,
} from 'lucide-react';
import { api } from '@/api/client';
import { PageTabs } from '@/components/page-tabs';
import { StatusChip } from '@/components/status-chip';
import { Button } from '@/components/ui/button';
import { Switch } from '@/components/ui/switch';
import { EmptyState } from '@/components/empty-state';
import { ErrorState } from '@/components/error-state';
import { FormDialog, ConfirmDialog, TextField, TextAreaField, SelectField } from '@/components/form-dialog';
import { useTabIndex } from '@/hooks/use-tab-param';
import {
  ACCENT,
  type Flag,
  FlagTypeBadge,
  RuleTypeBadge,
  formatDate,
  formatNumber,
} from './flags-shared';

const DETAIL_TABS = ['Settings', 'Rules', 'Overrides', 'Stats'];

export function FlagDetail({
  flag,
  onBack,
  onChange,
  onDeleted,
}: {
  flag: Flag;
  onBack: () => void;
  onChange: (flag: Flag) => void;
  onDeleted: () => void;
}) {
  const [tab, setTab] = useTabIndex(DETAIL_TABS);
  const flagId = String(flag.$id ?? '');
  const enabled = flag.enabled === true;

  return (
    <div className="flex flex-col gap-4 p-6 md:p-8">
      {/* Breadcrumb header */}
      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={onBack}
          className="flex cursor-pointer items-center gap-2 text-[length:var(--text-control)] text-text-muted transition-colors hover:text-text-secondary"
        >
          <ArrowLeft size={16} />
          Feature Flags
        </button>
        <ChevronRight size={14} className="text-text-subtle" />
        <h1 className="flex-1 truncate text-[length:var(--text-h1)] font-semibold text-text-primary">
          {String(flag.name ?? flag.key ?? '')}
        </h1>
        <StatusChip label={enabled ? 'enabled' : 'disabled'} />
      </div>

      {/* Key + type */}
      <div className="flex items-center gap-4">
        <span className="flex items-center gap-1.5 font-[family-name:var(--font-mono)] text-[length:var(--text-label)] text-text-muted">
          <Key size={12} className="text-text-subtle" />
          {String(flag.key ?? '')}
        </span>
        <FlagTypeBadge type={String(flag.type ?? 'boolean')} />
      </div>

      <PageTabs tabs={DETAIL_TABS} selected={tab} onChange={setTab} />

      {tab === 0 && <SettingsTab flag={flag} onChange={onChange} onDeleted={onDeleted} />}
      {tab === 1 && <RulesTab flagId={flagId} />}
      {tab === 2 && <OverridesTab flagId={flagId} />}
      {tab === 3 && <StatsTab flagId={flagId} />}
    </div>
  );
}

// --- Settings tab ----------------------------------------------------------

function DetailRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex items-start gap-4 py-3">
      <div className="w-36 shrink-0 text-[length:var(--text-body)] font-medium text-text-muted">{label}</div>
      <div
        className={`flex-1 break-words text-[length:var(--text-body)] text-text-primary ${
          mono ? 'font-[family-name:var(--font-mono)]' : ''
        }`}
      >
        {value}
      </div>
    </div>
  );
}

function SettingsTab({
  flag,
  onChange,
  onDeleted,
}: {
  flag: Flag;
  onChange: (flag: Flag) => void;
  onDeleted: () => void;
}) {
  const [editing, setEditing] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [toggling, setToggling] = useState(false);
  const flagId = String(flag.$id ?? '');
  const enabled = flag.enabled === true;
  const description = String(flag.description ?? '');

  const toggleEnabled = async (next: boolean) => {
    setToggling(true);
    try {
      const res = await api.patch(`/flags/${flagId}/toggle`, { enabled: next });
      onChange(res.data as Flag);
    } finally {
      setToggling(false);
    }
  };

  const deleteFlag = async () => {
    setDeleting(true);
    try {
      await api.delete(`/flags/${flagId}`);
      setConfirmDelete(false);
      onDeleted();
    } finally {
      setDeleting(false);
    }
  };

  return (
    <div className="flex flex-col gap-6">
      <div className="rounded-[var(--radius-10)] border border-border bg-surface p-5">
        <div className="mb-2 flex items-center justify-between">
          <div className="text-[length:var(--text-control)] font-semibold text-text-primary">Flag details</div>
          <Button variant="outline" size="sm" onClick={() => setEditing(true)}>
            <Pencil size={13} />
            Edit
          </Button>
        </div>
        <div className="divide-y divide-border">
          <DetailRow label="Key" value={String(flag.key ?? '')} mono />
          <DetailRow label="Name" value={String(flag.name ?? '')} />
          <DetailRow label="Description" value={description || 'No description'} />
          <DetailRow label="Type" value={String(flag.type ?? 'boolean')} />
          <DetailRow label="Default value" value={String(flag.defaultValue ?? 'N/A')} mono />
          <div className="flex items-center gap-4 py-3">
            <div className="w-36 shrink-0 text-[length:var(--text-body)] font-medium text-text-muted">Status</div>
            <div className="flex flex-1 items-center gap-3">
              <Switch checked={enabled} disabled={toggling} onCheckedChange={toggleEnabled} />
              <span className="text-[length:var(--text-body)] text-text-secondary">
                {enabled ? 'Enabled' : 'Disabled'}
              </span>
            </div>
          </div>
          <DetailRow label="Created" value={formatDate(flag.$createdAt)} />
          <DetailRow label="Updated" value={formatDate(flag.$updatedAt)} />
        </div>
      </div>

      {/* Danger zone */}
      <div className="flex items-center gap-4 rounded-[var(--radius-10)] border border-[color-mix(in_srgb,var(--color-danger)_40%,var(--border))] bg-surface p-5">
        <div className="flex-1">
          <div className="text-[length:var(--text-control)] font-medium text-[var(--status-danger)]">Delete flag</div>
          <div className="mt-1 text-[length:var(--text-body)] text-text-muted">
            Permanently delete this flag and all its rules and overrides.
          </div>
        </div>
        <Button variant="destructive" onClick={() => setConfirmDelete(true)}>
          Delete
        </Button>
      </div>

      {editing && (
        <EditFlagDialog flag={flag} onClose={() => setEditing(false)} onSaved={onChange} />
      )}

      <ConfirmDialog
        open={confirmDelete}
        onOpenChange={(o) => !o && setConfirmDelete(false)}
        title="Delete flag"
        message={`Permanently delete "${String(flag.name ?? flag.key ?? '')}" and all its rules and overrides? This cannot be undone.`}
        loading={deleting}
        onConfirm={deleteFlag}
      />
    </div>
  );
}

function EditFlagDialog({
  flag,
  onClose,
  onSaved,
}: {
  flag: Flag;
  onClose: () => void;
  onSaved: (flag: Flag) => void;
}) {
  const [name, setName] = useState(String(flag.name ?? ''));
  const [description, setDescription] = useState(String(flag.description ?? ''));
  const type = String(flag.type ?? 'boolean');
  const [defaultValue, setDefaultValue] = useState(String(flag.defaultValue ?? ''));
  const [saving, setSaving] = useState(false);

  const save = async () => {
    setSaving(true);
    try {
      // Full update uses PUT /flags/{id}. The backend updateFlag binds only
      // name/description/defaultValue/enabled/tags — `type` is not accepted,
      // so it's omitted here rather than sent and silently ignored.
      const res = await api.put(`/flags/${String(flag.$id)}`, {
        name: name.trim(),
        description: description.trim(),
        defaultValue: defaultValue.trim(),
        enabled: flag.enabled ?? false,
      });
      onSaved(res.data as Flag);
      onClose();
    } finally {
      setSaving(false);
    }
  };

  return (
    <FormDialog
      open
      onOpenChange={(o) => !o && onClose()}
      title="Edit flag"
      subtitle={String(flag.key ?? '')}
      submitLabel="Save"
      loading={saving}
      onSubmit={save}
    >
      <TextField label="Name" value={name} onChange={(e) => setName(e.target.value)} placeholder="Display name" autoFocus />
      <TextAreaField
        label="Description"
        value={description}
        onChange={(e) => setDescription(e.target.value)}
        placeholder="What does this flag control?"
        rows={2}
      />
      <TextField
        label="Type"
        value={type}
        onChange={() => {}}
        disabled
        hint="A flag's type is fixed when it is created."
      />
      <TextField
        label="Default value"
        value={defaultValue}
        onChange={(e) => setDefaultValue(e.target.value)}
        placeholder={defaultValueHint(type)}
      />
    </FormDialog>
  );
}

export function defaultValueHint(type: string): string {
  switch (type) {
    case 'boolean':
      return 'true or false';
    case 'number':
      return '0';
    case 'json':
      return '{}';
    default:
      return 'value';
  }
}

// --- Rules tab -------------------------------------------------------------

const RULE_TYPES = ['percentage', 'attribute', 'user', 'team'];
const RULE_OPERATORS = ['eq', 'neq', 'contains', 'starts_with', 'ends_with', 'in', 'not_in', 'gt', 'lt'];

function conditionsSummary(conditions: unknown): string {
  if (Array.isArray(conditions) && conditions.length > 0) {
    return conditions
      .map((c) => {
        const m = c as Record<string, unknown>;
        return `${String(m.attribute ?? '')} ${String(m.operator ?? '')} ${String(m.value ?? '')}`;
      })
      .join(', ');
  }
  if (typeof conditions === 'string' && conditions) return conditions;
  return 'No conditions';
}

function RulesTab({ flagId }: { flagId: string }) {
  const [adding, setAdding] = useState(false);
  const query = useQuery({
    queryKey: ['/flags', flagId, 'rules'],
    queryFn: async () => (await api.get(`/flags/${flagId}/rules`)).data as Record<string, unknown>,
  });
  const rules = (query.data?.rules as Record<string, unknown>[] | undefined) ?? [];

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-[length:var(--text-control)] font-medium text-text-primary">
          <GitBranch size={16} className="text-text-muted" />
          Targeting Rules
        </div>
        <Button size="sm" onClick={() => setAdding(true)}>
          <Plus size={14} />
          Add Rule
        </Button>
      </div>

      {query.error ? (
        <ErrorState error={query.error} onRetry={() => query.refetch()} />
      ) : query.isLoading ? (
        <div className="py-16 text-center text-[length:var(--text-body)] text-text-muted">Loading rules…</div>
      ) : rules.length === 0 ? (
        <EmptyState
          icon={GitBranch}
          title="No rules defined"
          subtitle="All users will receive the default value."
        />
      ) : (
        <div className="flex flex-col gap-3">
          {rules.map((rule) => (
            <RuleCard key={String(rule.$id)} flagId={flagId} rule={rule} onDeleted={() => query.refetch()} />
          ))}
        </div>
      )}

      {adding && (
        <AddRuleDialog flagId={flagId} onClose={() => setAdding(false)} onAdded={() => query.refetch()} />
      )}
    </div>
  );
}

function RuleCard({
  flagId,
  rule,
  onDeleted,
}: {
  flagId: string;
  rule: Record<string, unknown>;
  onDeleted: () => void;
}) {
  const [confirm, setConfirm] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const ruleType = String(rule.type ?? 'percentage');
  const rollout = Number(rule.rolloutPct ?? rule.rolloutPercentage ?? 100);

  const remove = async () => {
    setDeleting(true);
    try {
      await api.delete(`/flags/${flagId}/rules/${String(rule.$id)}`);
      setConfirm(false);
      onDeleted();
    } finally {
      setDeleting(false);
    }
  };

  return (
    <div className="rounded-[var(--radius-10)] border border-border bg-surface p-4">
      <div className="flex items-center gap-3">
        <RuleTypeBadge type={ruleType} />
        <span className="flex-1 truncate font-[family-name:var(--font-mono)] text-[length:var(--text-label)] text-text-muted">
          {conditionsSummary(rule.conditions)}
        </span>
        <button
          type="button"
          onClick={() => setConfirm(true)}
          className="rounded-[var(--radius-6)] p-1.5 text-text-muted transition-colors hover:bg-fill hover:text-[var(--color-danger)]"
          aria-label="Delete rule"
        >
          <Trash2 size={16} />
        </button>
      </div>
      <div className="mt-2 flex items-center gap-1.5 text-[length:var(--text-label)]">
        <ArrowRight size={12} className="text-text-subtle" />
        <span className="text-text-muted">Value:</span>
        <span className="font-[family-name:var(--font-mono)] text-text-primary">{String(rule.value ?? 'N/A')}</span>
      </div>
      <div className="mt-2 flex items-center gap-3 text-[length:var(--text-label)]">
        <Users size={12} className="text-text-subtle" />
        <span className="whitespace-nowrap text-text-muted">Rollout: {Math.round(rollout)}%</span>
        <div className="h-1.5 flex-1 overflow-hidden rounded-full" style={{ backgroundColor: 'color-mix(in srgb, var(--color-accent) 12%, transparent)' }}>
          <div className="h-full rounded-full" style={{ width: `${Math.max(0, Math.min(100, rollout))}%`, backgroundColor: ACCENT }} />
        </div>
      </div>

      <ConfirmDialog
        open={confirm}
        onOpenChange={(o) => !o && setConfirm(false)}
        title="Delete rule"
        message="Are you sure? This action cannot be undone."
        loading={deleting}
        onConfirm={remove}
      />
    </div>
  );
}

function AddRuleDialog({
  flagId,
  onClose,
  onAdded,
}: {
  flagId: string;
  onClose: () => void;
  onAdded: () => void;
}) {
  const [ruleType, setRuleType] = useState('percentage');
  const [attribute, setAttribute] = useState('');
  const [operator, setOperator] = useState('eq');
  const [condValue, setCondValue] = useState('');
  const [serveValue, setServeValue] = useState('');
  const [rollout, setRollout] = useState(100);
  const [saving, setSaving] = useState(false);

  const attributeHint =
    ruleType === 'user' ? 'userId' : ruleType === 'team' ? 'teamId' : 'e.g. email, country, plan';

  const save = async () => {
    setSaving(true);
    try {
      const conditions: Record<string, unknown>[] = [];
      if (ruleType !== 'percentage' && attribute.trim()) {
        const raw = condValue.trim();
        const value =
          operator === 'in' || operator === 'not_in' ? raw.split(',').map((s) => s.trim()) : raw;
        conditions.push({ attribute: attribute.trim(), operator, value });
      }
      await api.post(`/flags/${flagId}/rules`, {
        type: ruleType,
        conditions,
        value: serveValue.trim(),
        rolloutPct: Math.round(rollout),
      });
      onAdded();
      onClose();
    } finally {
      setSaving(false);
    }
  };

  return (
    <FormDialog
      open
      onOpenChange={(o) => !o && onClose()}
      title="Add rule"
      subtitle="Target a subset of users with a specific value"
      submitLabel="Add rule"
      loading={saving}
      onSubmit={save}
    >
      <SelectField
        label="Rule type"
        value={ruleType}
        onChange={setRuleType}
        options={RULE_TYPES.map((t) => ({ value: t, label: t }))}
      />
      {ruleType !== 'percentage' && (
        <>
          <TextField
            label="Attribute"
            value={attribute}
            onChange={(e) => setAttribute(e.target.value)}
            placeholder={attributeHint}
          />
          <SelectField
            label="Operator"
            value={operator}
            onChange={setOperator}
            options={RULE_OPERATORS.map((o) => ({ value: o, label: o }))}
          />
          <TextField
            label="Condition value"
            value={condValue}
            onChange={(e) => setCondValue(e.target.value)}
            placeholder="e.g. US  (comma-separate for in/not_in)"
          />
        </>
      )}
      <TextField
        label="Serve value"
        value={serveValue}
        onChange={(e) => setServeValue(e.target.value)}
        placeholder="Value returned when this rule matches"
      />
      <div className="flex flex-col gap-1.5">
        <div className="flex items-center justify-between">
          <span className="text-[length:var(--text-label)] font-medium text-text-muted">Rollout</span>
          <span className="text-[length:var(--text-body)] font-medium text-text-primary">{rollout}%</span>
        </div>
        <input
          type="range"
          min={0}
          max={100}
          value={rollout}
          onChange={(e) => setRollout(Number(e.target.value))}
          className="w-full cursor-pointer accent-[var(--color-accent)]"
        />
      </div>
    </FormDialog>
  );
}

// --- Overrides tab ---------------------------------------------------------

function OverridesTab({ flagId }: { flagId: string }) {
  const [adding, setAdding] = useState(false);
  const query = useQuery({
    queryKey: ['/flags', flagId, 'overrides'],
    queryFn: async () => (await api.get(`/flags/${flagId}/overrides`)).data as Record<string, unknown>,
  });
  const overrides = (query.data?.overrides as Record<string, unknown>[] | undefined) ?? [];

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-[length:var(--text-control)] font-medium text-text-primary">
          <UserCog size={16} className="text-text-muted" />
          Target Overrides
        </div>
        <Button size="sm" onClick={() => setAdding(true)}>
          <Plus size={14} />
          Add Override
        </Button>
      </div>

      {query.error ? (
        <ErrorState error={query.error} onRetry={() => query.refetch()} />
      ) : query.isLoading ? (
        <div className="py-16 text-center text-[length:var(--text-body)] text-text-muted">Loading overrides…</div>
      ) : overrides.length === 0 ? (
        <EmptyState
          icon={UserCog}
          title="No overrides set"
          subtitle="Override flag values for specific users or teams."
        />
      ) : (
        <div className="flex flex-col gap-2">
          {overrides.map((o) => (
            <OverrideRow
              key={`${String(o.targetType)}-${String(o.targetId)}`}
              flagId={flagId}
              override={o}
              onDeleted={() => query.refetch()}
            />
          ))}
        </div>
      )}

      {adding && (
        <AddOverrideDialog flagId={flagId} onClose={() => setAdding(false)} onAdded={() => query.refetch()} />
      )}
    </div>
  );
}

function OverrideRow({
  flagId,
  override,
  onDeleted,
}: {
  flagId: string;
  override: Record<string, unknown>;
  onDeleted: () => void;
}) {
  const [confirm, setConfirm] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const targetType = String(override.targetType ?? 'user');
  const isUser = targetType === 'user';
  const color = isUser ? '#0EA5E9' : '#F59E0B';

  const remove = async () => {
    setDeleting(true);
    try {
      await api.delete(`/flags/${flagId}/overrides/${targetType}/${String(override.targetId)}`);
      setConfirm(false);
      onDeleted();
    } finally {
      setDeleting(false);
    }
  };

  return (
    <div className="flex items-center gap-3 rounded-[var(--radius-10)] border border-border bg-surface px-4 py-3">
      {isUser ? (
        <User size={16} className="text-text-muted" />
      ) : (
        <Users size={16} className="text-text-muted" />
      )}
      <span
        className="rounded-[var(--radius-sm)] px-1.5 py-0.5 text-[length:var(--text-caption)]"
        style={{ color, backgroundColor: `color-mix(in srgb, ${color} 12%, transparent)` }}
      >
        {targetType}
      </span>
      <span className="flex-1 truncate font-[family-name:var(--font-mono)] text-[length:var(--text-label)] text-text-primary">
        {String(override.targetId ?? '')}
      </span>
      <ArrowRight size={14} className="text-text-subtle" />
      <span className="font-[family-name:var(--font-mono)] text-[length:var(--text-label)] font-medium" style={{ color: ACCENT }}>
        {String(override.value ?? 'N/A')}
      </span>
      <button
        type="button"
        onClick={() => setConfirm(true)}
        className="rounded-[var(--radius-6)] p-1.5 text-text-muted transition-colors hover:bg-fill hover:text-[var(--color-danger)]"
        aria-label="Remove override"
      >
        <Trash2 size={16} />
      </button>

      <ConfirmDialog
        open={confirm}
        onOpenChange={(o) => !o && setConfirm(false)}
        title="Remove override"
        message="Are you sure? This action cannot be undone."
        loading={deleting}
        onConfirm={remove}
      />
    </div>
  );
}

function AddOverrideDialog({
  flagId,
  onClose,
  onAdded,
}: {
  flagId: string;
  onClose: () => void;
  onAdded: () => void;
}) {
  const [targetType, setTargetType] = useState('user');
  const [targetId, setTargetId] = useState('');
  const [value, setValue] = useState('');
  const [saving, setSaving] = useState(false);

  const save = async () => {
    setSaving(true);
    try {
      await api.post(`/flags/${flagId}/overrides`, { targetType, targetId, value });
      onAdded();
      onClose();
    } finally {
      setSaving(false);
    }
  };

  return (
    <FormDialog
      open
      onOpenChange={(o) => !o && onClose()}
      title="Add Override"
      submitLabel="Add Override"
      loading={saving}
      submitDisabled={!targetId.trim()}
      onSubmit={save}
    >
      <SelectField
        label="Target Type"
        value={targetType}
        onChange={setTargetType}
        options={[
          { value: 'user', label: 'user' },
          { value: 'team', label: 'team' },
        ]}
      />
      <TextField
        label="Target ID"
        value={targetId}
        onChange={(e) => setTargetId(e.target.value)}
        placeholder={targetType === 'user' ? 'User ID' : 'Team ID'}
        autoFocus
      />
      <TextField
        label="Value"
        value={value}
        onChange={(e) => setValue(e.target.value)}
        placeholder="Override value for this target"
      />
    </FormDialog>
  );
}

// --- Stats tab -------------------------------------------------------------

function StatCard({ icon: Icon, label, value }: { icon: typeof BarChart3; label: string; value: string }) {
  return (
    <div className="flex-1 rounded-[var(--radius-10)] border border-border bg-surface p-5">
      <div className="flex items-center gap-2">
        <Icon size={16} style={{ color: ACCENT }} />
        <span className="text-[length:var(--text-label)] text-text-muted">{label}</span>
      </div>
      <div className="mt-3 text-[length:var(--text-h2)] font-semibold text-text-primary">{value}</div>
    </div>
  );
}

function StatsTab({ flagId }: { flagId: string }) {
  const query = useQuery({
    queryKey: ['/flags', flagId, 'stats'],
    queryFn: async () => (await api.get(`/flags/${flagId}/stats`)).data as Record<string, unknown>,
  });

  if (query.error) return <ErrorState error={query.error} onRetry={() => query.refetch()} />;
  if (query.isLoading)
    return <div className="py-16 text-center text-[length:var(--text-body)] text-text-muted">Loading stats…</div>;

  const data = query.data ?? {};
  const totalEvals = Number(data.totalEvaluations ?? 0);
  const uniqueUsers = data.uniqueUsers ?? 0;
  const distribution = (data.valueDistribution as Record<string, unknown> | undefined) ?? {};
  const entries = Object.entries(distribution);

  return (
    <div className="flex flex-col gap-6">
      <div className="flex gap-4">
        <StatCard icon={BarChart3} label="Total Evaluations" value={formatNumber(totalEvals)} />
        <StatCard icon={Users} label="Unique Users" value={formatNumber(uniqueUsers)} />
        <StatCard icon={PieChart} label="Distinct Values" value={String(entries.length)} />
      </div>

      <div className="rounded-[var(--radius-10)] border border-border bg-surface p-5">
        <div className="mb-4 flex items-center gap-2 text-[length:var(--text-control)] font-medium text-text-primary">
          <PieChart size={16} className="text-text-muted" />
          Value Distribution
        </div>
        {entries.length === 0 ? (
          <div className="py-6 text-center text-[length:var(--text-body)] text-text-subtle">No evaluation data yet</div>
        ) : (
          <div className="flex flex-col gap-3">
            {entries.map(([key, raw]) => {
              const count = Number(raw ?? 0);
              const pct = totalEvals > 0 ? (count / totalEvals) * 100 : 0;
              return (
                <div key={key} className="flex flex-col gap-1.5">
                  <div className="flex items-center justify-between text-[length:var(--text-body)]">
                    <span className="font-[family-name:var(--font-mono)] text-text-primary">{key}</span>
                    <span className="text-text-muted">
                      {count} ({pct.toFixed(1)}%)
                    </span>
                  </div>
                  <div
                    className="h-2 overflow-hidden rounded-full"
                    style={{ backgroundColor: 'color-mix(in srgb, var(--color-accent) 12%, transparent)' }}
                  >
                    <div className="h-full rounded-full" style={{ width: `${pct}%`, backgroundColor: ACCENT }} />
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
