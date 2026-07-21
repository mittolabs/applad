import { useEffect, useState } from 'react';
import { GitBranch, Upload } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { FormDialog, FormField, TextField } from '@/components/form-dialog';
import { toast } from '@/components/toast';
import { ChoiceChip } from '../sites/SiteDetail';
import { SourceDropzone, type PickedSource } from '../deploy-shared/SourceDropzone';
import { buildTarGz } from '@/lib/targz';

/*
 * Creating a suite is describing how the project runs its own tests: a base
 * image, an optional setup command, the test command, and where the JUnit
 * report lands. Applad needs nothing else to support a framework.
 */

// Enough to fill the form for the stacks people actually arrive with.
const PRESETS = [
  {
    label: 'Node',
    image: 'node:20-alpine',
    setupCmd: 'npm ci',
    command: 'npm test',
    reportPath: 'junit.xml',
  },
  {
    label: 'Python',
    image: 'python:3.12-slim',
    setupCmd: 'pip install -r requirements.txt',
    command: 'pytest --junitxml=junit.xml',
    reportPath: 'junit.xml',
  },
  {
    label: 'Go',
    image: 'golang:1.22-alpine',
    setupCmd: 'go install gotest.tools/gotestsum@latest',
    command: 'gotestsum --junitfile junit.xml ./...',
    reportPath: 'junit.xml',
  },
];

export function CreateSuiteDialog({
  open,
  onOpenChange,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (o: boolean) => void;
  onCreated: () => void;
}) {
  const [name, setName] = useState('');
  const [sourceType, setSourceType] = useState<'git' | 'upload'>('upload');
  const [sourceUrl, setSourceUrl] = useState('');
  const [branch, setBranch] = useState('main');
  const [image, setImage] = useState(PRESETS[0].image);
  const [setupCmd, setSetupCmd] = useState(PRESETS[0].setupCmd);
  const [command, setCommand] = useState(PRESETS[0].command);
  const [reportPath, setReportPath] = useState(PRESETS[0].reportPath);
  const [source, setSource] = useState<PickedSource | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!open) return;
    setName('');
    setSource(null);
    setBusy(false);
  }, [open]);

  const applyPreset = (p: (typeof PRESETS)[number]) => {
    setImage(p.image);
    setSetupCmd(p.setupCmd);
    setCommand(p.command);
    setReportPath(p.reportPath);
  };

  const create = async () => {
    setBusy(true);
    try {
      const res = await api.post('/tests/suites', {
        name: name.trim(),
        sourceType,
        sourceUrl: sourceType === 'git' ? sourceUrl.trim() : '',
        branch: sourceType === 'git' ? branch.trim() : '',
        image: image.trim(),
        setupCmd: setupCmd.trim(),
        command: command.trim(),
        reportPath: reportPath.trim(),
      });
      const suiteId = (res.data as { $id: string }).$id;

      if (sourceType === 'upload' && source) {
        let body: Blob;
        if (source.archive) {
          body = source.archive;
        } else {
          const entries = await Promise.all(
            source.files!.map(async (f) => ({
              path: f.path,
              data: new Uint8Array(await f.file.arrayBuffer()),
            })),
          );
          body = await buildTarGz(entries);
        }
        await api.post(`/tests/suites/${suiteId}/source`, body, {
          headers: { 'Content-Type': 'application/octet-stream' },
        });
      }

      toast.success(`Created ${name.trim()}`);
      onOpenChange(false);
      onCreated();
    } catch (e) {
      toast.error(friendlyError(e));
      setBusy(false);
    }
  };

  return (
    <FormDialog
      open={open}
      onOpenChange={onOpenChange}
      title="New test suite"
      subtitle="How this project runs its tests."
      submitLabel="Create"
      loading={busy}
      submitDisabled={!name.trim() || !command.trim() || (sourceType === 'upload' ? !source : !sourceUrl.trim())}
      onSubmit={create}
      width={540}
    >
      <TextField
        label="Name"
        value={name}
        onChange={(e) => setName(e.target.value)}
        placeholder="unit tests"
        autoFocus
      />

      <FormField label="Stack">
        <div className="flex flex-wrap gap-1.5">
          {PRESETS.map((p) => (
            <button
              key={p.label}
              type="button"
              onClick={() => applyPreset(p)}
              className="rounded-[var(--radius-6)] border border-border px-2.5 py-1 text-[length:var(--text-caption)] text-text-muted transition-colors hover:text-text-primary"
            >
              {p.label}
            </button>
          ))}
        </div>
        <p className="mt-1.5 text-[length:var(--text-caption)] text-text-subtle">
          Fills the fields below. Any stack works as long as it can write a JUnit report.
        </p>
      </FormField>

      <FormField label="Source">
        <div className="flex gap-2">
          <ChoiceChip icon={GitBranch} label="Git repository" selected={sourceType === 'git'} onClick={() => setSourceType('git')} />
          <ChoiceChip icon={Upload} label="Upload" selected={sourceType === 'upload'} onClick={() => setSourceType('upload')} />
        </div>
      </FormField>

      {sourceType === 'git' ? (
        <>
          <TextField label="Repository URL" value={sourceUrl} onChange={(e) => setSourceUrl(e.target.value)} placeholder="https://github.com/user/repo" />
          <TextField label="Branch" value={branch} onChange={(e) => setBranch(e.target.value)} placeholder="main" />
        </>
      ) : (
        <SourceDropzone value={source} onChange={setSource} />
      )}

      <TextField label="Image" value={image} onChange={(e) => setImage(e.target.value)} placeholder="node:20-alpine" />
      <TextField label="Setup command" value={setupCmd} onChange={(e) => setSetupCmd(e.target.value)} placeholder="npm ci" />
      <TextField label="Test command" value={command} onChange={(e) => setCommand(e.target.value)} placeholder="npm test" />
      <TextField
        label="Report path"
        value={reportPath}
        onChange={(e) => setReportPath(e.target.value)}
        placeholder="junit.xml"
        hint="Where the suite writes its JUnit XML, relative to the project root. This is how results are read."
      />
    </FormDialog>
  );
}
