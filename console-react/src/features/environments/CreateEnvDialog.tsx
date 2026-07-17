import { useEffect, useState } from 'react';
import { api, friendlyError } from '@/api/client';
import { toast } from '@/components/toast';
import { FormDialog, TextField } from '@/components/form-dialog';

/* Create a new deploy environment — ports _showCreateEnvDialog. */
export function CreateEnvDialog({
  open,
  onOpenChange,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: (id: string) => void;
}) {
  const [name, setName] = useState('');
  const [slug, setSlug] = useState('');
  const [branch, setBranch] = useState('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (open) {
      setName('');
      setSlug('');
      setBranch('');
      setLoading(false);
    }
  }, [open]);

  const create = async () => {
    setLoading(true);
    try {
      const res = await api.post('/deploy/environments', {
        name: name.trim(),
        slug: slug.trim() || name.trim().toLowerCase().replace(/ /g, '-'),
        branch: branch.trim(),
      });
      const id = String((res.data as Record<string, unknown>)['$id'] ?? '');
      toast.success('Environment created');
      onOpenChange(false);
      if (id) onCreated(id);
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
      title="New environment"
      submitLabel="Create"
      loading={loading}
      submitDisabled={!name.trim()}
      onSubmit={create}
    >
      <TextField
        label="Name"
        placeholder="e.g. Staging"
        value={name}
        onChange={(e) => setName(e.target.value)}
        autoFocus
      />
      <TextField
        label="Slug"
        placeholder="e.g. staging"
        value={slug}
        onChange={(e) => setSlug(e.target.value)}
      />
      <TextField
        label="Branch"
        placeholder="e.g. main (optional)"
        value={branch}
        onChange={(e) => setBranch(e.target.value)}
      />
    </FormDialog>
  );
}
