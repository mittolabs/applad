import { FlaskConical } from 'lucide-react';
import { StandaloneLayout } from '@/shell/StandaloneLayout';
import { Switch } from '@/components/ui/switch';
import { Button } from '@/components/ui/button';
import {
  useExperimentsStore,
  type ExperimentKey,
} from '@/stores/experiments';

/* Ports experiments/experiments_page.dart — console-level feature toggles
 * persisted to localStorage. Gates things like the AI chat button. */

const EXPERIMENTS: { key: ExperimentKey; label: string; description: string }[] = [
  { key: 'aiChat', label: 'AI Assistant', description: 'Show the in-console AI chat panel.' },
  { key: 'search', label: 'Advanced Search', description: 'Full-text search across resources.' },
  { key: 'analytics', label: 'Analytics', description: 'Product analytics dashboards.' },
  { key: 'cache', label: 'Edge Cache', description: 'Response caching controls.' },
  { key: 'edgeFunctions', label: 'Edge Functions', description: 'Run functions at the edge.' },
  { key: 'vectors', label: 'Vectors', description: 'Vector storage and search.' },
  { key: 'regions', label: 'Regions', description: 'Multi-region deployment.' },
];

export function ExperimentsPage() {
  const { flags, toggle, enableAll, disableAll } = useExperimentsStore();
  return (
    <StandaloneLayout showOrg={false}>
      <div className="mx-auto w-full max-w-3xl flex-1 px-6 py-8">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-[var(--radius)] bg-fill text-[var(--color-accent)]">
            <FlaskConical size={20} />
          </div>
          <div className="flex-1">
            <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">
              Experiments
            </h1>
            <div className="text-[length:var(--text-body)] text-text-muted">
              Toggle preview features. Stored locally in this browser.
            </div>
          </div>
          <Button variant="ghost" size="sm" onClick={disableAll}>
            Disable all
          </Button>
          <Button variant="secondary" size="sm" onClick={enableAll}>
            Enable all
          </Button>
        </div>

        <div className="mt-6 overflow-hidden rounded-[var(--radius-10)] border border-border">
          {EXPERIMENTS.map((e) => (
            <div
              key={e.key}
              className="flex items-center gap-4 border-b border-[var(--fill)] px-4 py-3.5 last:border-0"
            >
              <div className="flex-1">
                <div className="text-[length:var(--text-control)] font-medium text-text-primary">
                  {e.label}
                </div>
                <div className="text-[length:var(--text-caption)] text-text-muted">
                  {e.description}
                </div>
              </div>
              <Switch checked={flags[e.key]} onCheckedChange={() => toggle(e.key)} />
            </div>
          ))}
        </div>
      </div>
    </StandaloneLayout>
  );
}
