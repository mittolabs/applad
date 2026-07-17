import { useEffect, useState } from 'react';
import { GitBranch, Upload, UploadCloud, Info } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { FormField, TextField } from '@/components/form-dialog';
import { toast } from '@/components/toast';
import { ChoiceChip } from './SiteDetail';
import { FRAMEWORKS, frameworkById } from '../deploy-shared/frameworks';

/* Multi-step "Create site" form — ports _showCreateSiteForm in sites_page.dart
 * (Configuration → Source → Build). Prefilled from a DeployCreateEntry result. */

export interface SitePrefill {
  name: string;
  repository: string;
  branch: string;
  framework: string;
  sourceType: 'git' | 'upload';
  templateId?: string;
}

const STEP_LABELS = ['Configuration', 'Source', 'Build'];

export function CreateSiteDialog({
  open,
  onOpenChange,
  prefill,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (o: boolean) => void;
  prefill: SitePrefill | null;
  onCreated: () => void;
}) {
  const [step, setStep] = useState(0);
  const [name, setName] = useState('');
  const [framework, setFramework] = useState('nextjs');
  const [sourceType, setSourceType] = useState<'git' | 'upload'>('git');
  const [repository, setRepository] = useState('');
  const [branch, setBranch] = useState('main');
  const [installCommand, setInstallCommand] = useState('');
  const [buildCommand, setBuildCommand] = useState('');
  const [outputDirectory, setOutputDirectory] = useState('');
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    if (open && prefill) {
      setStep(0);
      setName(prefill.name);
      setFramework(prefill.framework);
      setSourceType(prefill.sourceType);
      setRepository(prefill.repository);
      setBranch(prefill.branch);
      setInstallCommand('');
      setBuildCommand('');
      setOutputDirectory('');
      setCreating(false);
    }
  }, [open, prefill]);

  const fw = frameworkById(framework);

  const goStep = (next: number) => {
    if (next === 2) {
      // Auto-fill build config from framework when empty.
      setInstallCommand((v) => v || fw.installCommand);
      setBuildCommand((v) => v || fw.buildCommand);
      setOutputDirectory((v) => v || fw.outputDir);
    }
    setStep(next);
  };

  const create = async () => {
    setCreating(true);
    try {
      await api.post('/deploy/targets', {
        name: name.trim(),
        type: 'web',
        framework,
        source: sourceType,
        repository: repository.trim(),
        branch: branch.trim(),
        buildCommand: buildCommand.trim(),
        outputDirectory: outputDirectory.trim(),
        installCommand: installCommand.trim(),
        ...(prefill?.templateId ? { templateId: prefill.templateId } : {}),
      });
      onOpenChange(false);
      onCreated();
    } catch (e) {
      toast.error(friendlyError(e));
      setCreating(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent width={540}>
        <DialogHeader>
          <DialogTitle>Create site</DialogTitle>
          <DialogDescription>{`Step ${step + 1} of 3: ${STEP_LABELS[step]}`}</DialogDescription>
        </DialogHeader>

        <div className="flex gap-1 px-6">
          {[0, 1, 2].map((i) => (
            <div
              key={i}
              className="h-[3px] flex-1 rounded-full"
              style={{ backgroundColor: i <= step ? 'var(--color-accent)' : 'var(--border)' }}
            />
          ))}
        </div>

        <DialogBody>
          {step === 0 && (
            <div className="flex flex-col gap-4">
              <TextField label="Site name" value={name} onChange={(e) => setName(e.target.value)} placeholder="my-awesome-site" autoFocus />
              <FormField label="Framework">
                <div className="flex flex-wrap gap-2">
                  {FRAMEWORKS.map((f) => {
                    const Icon = f.icon;
                    const selected = f.id === framework;
                    return (
                      <button
                        key={f.id}
                        type="button"
                        onClick={() => setFramework(f.id)}
                        className="flex w-[100px] flex-col items-center gap-1.5 rounded-[var(--radius)] border py-3 text-[length:var(--text-caption)] transition-colors"
                        style={
                          selected
                            ? {
                                borderColor: 'color-mix(in srgb, var(--color-accent) 40%, transparent)',
                                backgroundColor: 'color-mix(in srgb, var(--color-accent) 10%, transparent)',
                                color: 'var(--color-accent)',
                              }
                            : { borderColor: 'var(--border)', color: 'var(--text-muted)' }
                        }
                      >
                        <Icon size={24} />
                        {f.label}
                      </button>
                    );
                  })}
                </div>
              </FormField>
            </div>
          )}

          {step === 1 && (
            <div className="flex flex-col gap-4">
              <FormField label="Source">
                <div className="flex gap-2">
                  <ChoiceChip icon={GitBranch} label="Git repository" selected={sourceType === 'git'} onClick={() => setSourceType('git')} />
                  <ChoiceChip icon={Upload} label="Manual upload" selected={sourceType === 'upload'} onClick={() => setSourceType('upload')} />
                </div>
              </FormField>
              {sourceType === 'git' ? (
                <>
                  <TextField label="Repository URL" value={repository} onChange={(e) => setRepository(e.target.value)} placeholder="https://github.com/user/repo" />
                  <TextField label="Branch" value={branch} onChange={(e) => setBranch(e.target.value)} placeholder="main" />
                </>
              ) : (
                <div className="flex h-[120px] flex-col items-center justify-center gap-2 rounded-[var(--radius)] border border-field-border bg-fill text-text-muted">
                  <UploadCloud size={32} className="text-text-subtle" />
                  <span className="text-center text-[length:var(--text-label)]">
                    Drag &amp; drop your build output
                    <br />
                    or click to browse
                  </span>
                </div>
              )}
            </div>
          )}

          {step === 2 && (
            <div className="flex flex-col gap-4">
              <div className="flex items-center gap-2 rounded-[var(--radius)] border border-[color-mix(in_srgb,var(--color-accent)_15%,transparent)] bg-[color-mix(in_srgb,var(--color-accent)_8%,transparent)] p-3 text-[length:var(--text-label)] text-[var(--color-accent)]">
                <Info size={14} />
                {`Auto-detected from ${fw.label}. Edit if needed.`}
              </div>
              <TextField label="Install command" value={installCommand} onChange={(e) => setInstallCommand(e.target.value)} placeholder="npm install" />
              <TextField label="Build command" value={buildCommand} onChange={(e) => setBuildCommand(e.target.value)} placeholder="npm run build" />
              <TextField label="Output directory" value={outputDirectory} onChange={(e) => setOutputDirectory(e.target.value)} placeholder="dist" />
            </div>
          )}
        </DialogBody>

        <DialogFooter>
          {step > 0 ? (
            <Button variant="ghost" onClick={() => setStep(step - 1)}>
              Back
            </Button>
          ) : (
            <Button variant="ghost" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
          )}
          {step < 2 ? (
            <Button onClick={() => goStep(step + 1)} disabled={step === 0 && !name.trim()}>
              Next
            </Button>
          ) : (
            <Button loading={creating} onClick={create}>
              Create
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
