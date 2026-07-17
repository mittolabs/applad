import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Boxes } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { useOrgStore } from '@/stores/org';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';

/* Ports onboarding_page.dart — a single-step "create your organization" screen.
 * A freshly signed-up user has no org yet; this creates the first one and drops
 * them into its projects. */
export function OnboardingPage() {
  const navigate = useNavigate();
  const qc = useQueryClient();
  const { setCurrentOrg } = useOrgStore();
  const [name, setName] = useState('');
  const [error, setError] = useState<string | null>(null);

  const create = useMutation({
    mutationFn: async () => {
      const res = await api.post('/organizations', { name });
      return res.data as { $id?: string; id?: string };
    },
    onSuccess: (o) => {
      qc.invalidateQueries({ queryKey: ['organizations'] });
      const id = String(o.$id ?? o.id ?? '');
      if (id) {
        setCurrentOrg(id);
        navigate(`/org/${id}/projects`);
      } else {
        navigate('/projects');
      }
    },
    onError: (e) => setError(friendlyError(e)),
  });

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-6">
      <div className="w-full max-w-md">
        <div className="mb-6 flex items-center gap-2">
          <div className="flex h-9 w-9 items-center justify-center rounded-[var(--radius)] bg-[var(--color-accent)] text-white">
            <Boxes size={18} />
          </div>
          <span className="text-[length:var(--text-title)] font-semibold text-text-primary">
            Welcome to Applad
          </span>
        </div>

        <div className="rounded-[var(--radius-12)] border border-border bg-surface p-6">
          <h1 className="text-[length:var(--text-title)] font-semibold text-text-primary">
            Create your organization
          </h1>
          <p className="mt-1 text-[length:var(--text-body)] text-text-muted">
            Create your organization to get started. Organizations help you manage projects and
            team members.
          </p>

          <form
            onSubmit={(e) => {
              e.preventDefault();
              setError(null);
              if (name.trim()) create.mutate();
            }}
            className="mt-5 flex flex-col gap-4"
          >
            <div className="flex flex-col gap-1.5">
              <Label>Organization name</Label>
              <Input
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="Acme Inc."
                autoFocus
              />
            </div>

            {error && (
              <div className="rounded-[var(--radius)] bg-[color-mix(in_srgb,var(--color-danger)_10%,transparent)] px-3 py-2 text-[length:var(--text-caption)] text-[var(--status-danger)]">
                {error}
              </div>
            )}

            <Button
              type="submit"
              className="w-full"
              loading={create.isPending}
              disabled={!name.trim()}
            >
              Create organization
            </Button>
          </form>
        </div>
      </div>
    </div>
  );
}
