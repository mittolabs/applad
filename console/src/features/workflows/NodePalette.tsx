import { useState } from 'react';
import { ArrowLeft, ChevronRight, Plus, Search, X, Zap } from 'lucide-react';
import {
  ALL_NODE_DEFS,
  CATEGORIES,
  PALETTE_SECTIONS,
  type NodeCategory,
  type NodeDef,
} from './nodeDefs';

export function NodePalette({
  onAdd,
  onClose,
}: {
  onAdd: (type: string) => void;
  onClose: () => void;
}) {
  const [category, setCategory] = useState<string | null>(null);
  const [query, setQuery] = useState('');
  const q = query.trim().toLowerCase();
  const isSearch = q.length > 0;

  const pick = (type: string) => {
    onAdd(type);
    setCategory(null);
    setQuery('');
  };

  const categoryDef = (name: string): NodeCategory =>
    CATEGORIES.find((c) => c.name === name) ?? CATEGORIES[0];

  const HeaderIcon = category && !isSearch ? categoryDef(category).icon : null;

  return (
    <div className="flex h-full w-[300px] shrink-0 flex-col border-l border-border bg-surface">
      {/* Header */}
      <div className="flex items-center gap-2 px-4 pt-4">
        {category && !isSearch && (
          <button
            type="button"
            onClick={() => setCategory(null)}
            className="text-text-secondary hover:text-text-primary"
          >
            <ArrowLeft size={16} />
          </button>
        )}
        {HeaderIcon && <HeaderIcon size={15} className="text-text-secondary" />}
        <span className="flex-1 text-[length:var(--text-control)] font-semibold text-text-primary">
          {isSearch ? 'Search results' : (category ?? 'What happens next?')}
        </span>
        <button
          type="button"
          onClick={onClose}
          className="text-text-secondary hover:text-text-primary"
        >
          <X size={16} />
        </button>
      </div>

      {/* Search */}
      <div className="px-4 pb-2 pt-3">
        <div className="flex items-center gap-2 rounded-[var(--radius)] border border-border bg-fill px-2.5 py-2">
          <Search size={15} className="text-text-subtle" />
          <input
            autoFocus
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search nodes..."
            className="w-full bg-transparent text-[length:var(--text-body)] text-text-primary outline-none placeholder:text-text-subtle"
          />
        </div>
      </div>
      <div className="h-px bg-border" />

      {/* Content */}
      <div className="min-h-0 flex-1 overflow-y-auto p-2">
        {isSearch ? (
          <SearchResults query={q} onPick={pick} />
        ) : category === null ? (
          <TopLevel onCategory={setCategory} />
        ) : (
          <CategoryContent category={category} onPick={pick} />
        )}
      </div>
    </div>
  );
}

function TopLevel({ onCategory }: { onCategory: (name: string) => void }) {
  return (
    <div>
      {CATEGORIES.map((c) => (
        <CategoryRow key={c.name} category={c} onClick={() => onCategory(c.name)} />
      ))}
      <div className="mt-2 px-3">
        <button
          type="button"
          onClick={() => onCategory('Triggers')}
          className="flex w-full items-center gap-2.5 rounded-[var(--radius)] bg-fill px-3 py-2.5 text-left"
        >
          <Zap size={15} className="text-text-secondary" />
          <div className="flex-1">
            <div className="text-[length:var(--text-body)] text-text-primary">Add another trigger</div>
            <div className="text-[length:var(--text-caption)] text-text-subtle">
              Workflows can have multiple triggers
            </div>
          </div>
          <ChevronRight size={14} className="text-text-subtle" />
        </button>
      </div>
    </div>
  );
}

function CategoryRow({ category, onClick }: { category: NodeCategory; onClick: () => void }) {
  const Icon = category.icon;
  return (
    <button
      type="button"
      onClick={onClick}
      className="mb-0.5 flex w-full items-center gap-3 rounded-[var(--radius)] px-3 py-2.5 text-left transition-colors hover:bg-fill"
    >
      <Icon size={16} className="text-text-secondary" />
      <div className="flex-1">
        <div className="text-[length:var(--text-body)] text-text-primary">{category.name}</div>
        <div className="text-[length:var(--text-caption)] text-text-subtle">{category.description}</div>
      </div>
      <ChevronRight size={14} className="text-text-subtle" />
    </button>
  );
}

function CategoryContent({
  category,
  onPick,
}: {
  category: string;
  onPick: (type: string) => void;
}) {
  const sections = PALETTE_SECTIONS[category];
  const byType = (t: string) => ALL_NODE_DEFS.find((d) => d.type === t);

  if (sections) {
    return (
      <div>
        {sections.map((s) => (
          <div key={s.title}>
            <SectionLabel title={s.title} />
            {s.types
              .map(byType)
              .filter((d): d is NodeDef => !!d)
              .map((d) => (
                <NodeRow key={`${s.title}-${d.type}`} def={d} onClick={() => onPick(d.type)} />
              ))}
          </div>
        ))}
      </div>
    );
  }

  const items = ALL_NODE_DEFS.filter((d) => d.category === category);
  return (
    <div>
      {items.map((d) => (
        <NodeRow key={d.type} def={d} onClick={() => onPick(d.type)} />
      ))}
    </div>
  );
}

function SearchResults({ query, onPick }: { query: string; onPick: (type: string) => void }) {
  const matches = ALL_NODE_DEFS.filter(
    (d) =>
      d.label.toLowerCase().includes(query) ||
      d.description.toLowerCase().includes(query) ||
      d.type.toLowerCase().includes(query) ||
      d.category.toLowerCase().includes(query),
  );
  if (matches.length === 0) {
    return (
      <div className="p-8 text-center text-[length:var(--text-body)] text-text-subtle">
        No nodes found
      </div>
    );
  }
  return (
    <div>
      {matches.map((d) => (
        <NodeRow key={d.type} def={d} onClick={() => onPick(d.type)} />
      ))}
    </div>
  );
}

function SectionLabel({ title }: { title: string }) {
  return (
    <div className="px-3 pb-1 pt-3 text-[length:var(--text-caption)] font-semibold text-text-subtle">
      {title}
    </div>
  );
}

function NodeRow({ def, onClick }: { def: NodeDef; onClick: () => void }) {
  const Icon = def.icon;
  return (
    <button
      type="button"
      onClick={onClick}
      className="mb-0.5 flex w-full items-center gap-3 rounded-[var(--radius)] px-3 py-2 text-left transition-colors hover:bg-fill"
    >
      <div
        className="flex h-8 w-8 shrink-0 items-center justify-center rounded-[var(--radius-7)]"
        style={{ background: `color-mix(in srgb, ${def.color} 12%, transparent)`, color: def.color }}
      >
        <Icon size={15} />
      </div>
      <div className="min-w-0 flex-1">
        <div className="truncate text-[length:var(--text-body)] text-text-primary">{def.label}</div>
        <div className="truncate text-[length:var(--text-caption)] text-text-subtle">
          {def.description}
        </div>
      </div>
      <Plus size={14} className="text-text-subtle" />
    </button>
  );
}
