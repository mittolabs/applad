import { useEffect, useState } from 'react';
import { GitBranch, Upload, Info } from 'lucide-react';
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
import { SourceDropzone, type PickedSource, type Detection } from '../deploy-shared/SourceDropzone';
import { FrameworkLogo } from '../deploy-shared/FrameworkLogo';
import { buildTarGz } from '@/lib/targz';

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
  const [source, setSource] = useState<PickedSource | null>(null);
  const [detected, setDetected] = useState<Detection | null>(null);
  const [creating, setCreating] = useState(false);
  const [progress, setProgress] = useState('');

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
      setSource(null);
      setDetected(null);
      setCreating(false);
      setProgress('');
    }
  }, [open, prefill]);

  const fw = frameworkById(framework);

  const goStep = (next: number) => {
    if (next === 2) {
      if (sourceType === 'upload' && detected) {
        // Detection inspected the actual files, so trust it over the framework
        // picker: pre-built output yields no build command, sources yield one.
        setInstallCommand(detected.installCommand);
        setBuildCommand(detected.buildCommand);
        setOutputDirectory(detected.outputDir || '.');
      } else if (sourceType === 'upload') {
        // An archive we could not inspect. A manual upload IS the build
        // output, so assume it needs no build rather than guessing one that
        // fails on anything without a package.json.
        setInstallCommand('');
        setBuildCommand('');
        setOutputDirectory((v) => v || '.');
      } else {
        // Auto-fill build config from framework when empty.
        setInstallCommand((v) => v || fw.installCommand);
        setBuildCommand((v) => v || fw.buildCommand);
        setOutputDirectory((v) => v || fw.outputDir);
      }
    }
    setStep(next);
  };

  const create = async () => {
    setCreating(true);
    try {
      // A deployable site is three records: the target (what it is), the
      // pipeline (how it builds), and a release (an actual build). Creating
      // only the target left a site that could never deploy.
      setProgress('Creating site...');
      const target = await api.post('/deploy/targets', {
        name: name.trim(),
        type: 'web',
      });
      const targetId = target.data.$id ?? target.data.id;

      setProgress('Configuring build...');
      const pipeline = await api.post('/deploy/pipelines', {
        targetId,
        name: 'production',
        sourceType,
        sourceUrl: sourceType === 'git' ? repository.trim() : '',
        branch: sourceType === 'git' ? branch.trim() : '',
        buildCmd: buildCommand.trim(),
        outputDir: outputDirectory.trim() || '.',
      });
      const pipelineId = pipeline.data.$id ?? pipeline.data.id;

      if (sourceType === 'upload') {
        let body: Blob;
        if (source!.archive) {
          body = source!.archive;
        } else {
          setProgress('Packaging files...');
          const entries = await Promise.all(
            source!.files!.map(async (f) => ({
              path: f.path,
              data: new Uint8Array(await f.file.arrayBuffer()),
            })),
          );
          body = await buildTarGz(entries);
        }

        await api.post(`/deploy/pipelines/${pipelineId}/source`, body, {
          headers: { 'Content-Type': 'application/octet-stream' },
          onUploadProgress: (e) => {
            const pct = e.total ? Math.round((e.loaded / e.total) * 100) : 0;
            setProgress(`Uploading source... ${pct}%`);
          },
        });
      }

      setProgress('Starting deployment...');
      await api.post(`/deploy/pipelines/${pipelineId}/trigger`, {
        triggerType: 'manual',
        actor: 'console',
      });

      toast.success(`Deploying ${name.trim()}`);
      onOpenChange(false);
      onCreated();
    } catch (e) {
      toast.error(friendlyError(e));
      setCreating(false);
      setProgress('');
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
                <div className="grid grid-cols-4 gap-2">
                  {FRAMEWORKS.map((f) => {
                    const selected = f.id === framework;
                    return (
                      <button
                        key={f.id}
                        type="button"
                        onClick={() => setFramework(f.id)}
                        className="group flex flex-col items-center gap-1.5 rounded-[var(--radius)] border py-3 text-[length:var(--text-caption)] transition-colors"
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
                        <FrameworkLogo framework={f.id} size={24} active={selected} />
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
                <SourceDropzone
                  value={source}
                  onChange={setSource}
                  onDetected={(d) => {
                    setDetected(d);
                    // Detection knows what the source actually is; the picker
                    // on step 1 was only ever a guess.
                    if (d) setFramework(d.framework === 'docker' ? 'static' : d.framework);
                  }}
                />
              )}
            </div>
          )}

          {step === 2 && (
            <div className="flex flex-col gap-4">
              <div className="flex items-center gap-2 rounded-[var(--radius)] border border-[color-mix(in_srgb,var(--color-accent)_15%,transparent)] bg-[color-mix(in_srgb,var(--color-accent)_8%,transparent)] p-3 text-[length:var(--text-label)] text-[var(--color-accent)]">
                <Info size={14} />
                {detected
                  ? `Detected ${detected.framework} from ${detected.reason}. Edit if needed.`
                  : sourceType === 'upload'
                    ? 'Your upload is served as-is. Add a build command only if you uploaded sources.'
                    : `Auto-detected from ${fw.label}. Edit if needed.`}
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
            <Button
              onClick={() => goStep(step + 1)}
              disabled={
                (step === 0 && !name.trim()) ||
                (step === 1 && (sourceType === 'upload' ? !source : !repository.trim()))
              }
            >
              Next
            </Button>
          ) : (
            <>
              {progress && (
                <span className="mr-auto text-[length:var(--text-caption)] text-text-muted">{progress}</span>
              )}
              <Button
                loading={creating}
                onClick={create}
                disabled={sourceType === 'upload' ? !source : !repository.trim()}
              >
                Create
              </Button>
            </>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
