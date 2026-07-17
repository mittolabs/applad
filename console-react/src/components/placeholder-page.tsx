import { Construction } from 'lucide-react';

/* Temporary page shown for routes whose feature hasn't been ported yet.
 * Replaced one-by-one as Phase A–E features land. */
export function PlaceholderPage({ name }: { name: string }) {
  return (
    <div className="flex flex-col gap-6 p-6 md:p-8">
      <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">{name}</h1>
      <div className="flex flex-col items-center justify-center gap-3 rounded-[var(--radius-12)] border border-dashed border-border py-20 text-center">
        <Construction size={28} className="text-text-subtle" />
        <div className="text-[length:var(--text-body)] text-text-muted">
          {name} — coming from the React port
        </div>
      </div>
    </div>
  );
}
