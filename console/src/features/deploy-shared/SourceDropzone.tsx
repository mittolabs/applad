import { useRef, useState } from 'react';
import { UploadCloud, FileArchive, FolderOpen, X } from 'lucide-react';

/*
 * Source picker for manual-upload deploys. Accepts a dropped or browsed
 * folder (the common case: "here is my build output") as well as a prebuilt
 * .tar.gz / .zip archive.
 */

export interface PickedSource {
  /** Files to archive, or null when the user supplied an archive directly. */
  files: { path: string; file: File }[] | null;
  /** A .tar.gz or .zip the user picked, uploaded as-is. */
  archive: File | null;
  label: string;
  bytes: number;
}

const ARCHIVE_RE = /\.(tar\.gz|tgz|zip)$/i;

// Junk that should never reach a build context.
const IGNORED = new Set(['.DS_Store', 'Thumbs.db', '.git', 'node_modules', '.next']);

function ignored(path: string): boolean {
  return path.split('/').some((segment) => IGNORED.has(segment));
}

/** Walks a dropped directory entry into a flat file list. */
async function walkEntry(entry: FileSystemEntry, prefix: string): Promise<{ path: string; file: File }[]> {
  if (ignored(entry.name)) return [];

  if (entry.isFile) {
    const file = await new Promise<File>((resolve, reject) =>
      (entry as FileSystemFileEntry).file(resolve, reject),
    );
    return [{ path: prefix + entry.name, file }];
  }

  const reader = (entry as FileSystemDirectoryEntry).createReader();
  const children: FileSystemEntry[] = [];
  // readEntries returns at most 100 per call, so keep reading until it's dry.
  for (;;) {
    const batch = await new Promise<FileSystemEntry[]>((resolve, reject) =>
      reader.readEntries(resolve, reject),
    );
    if (batch.length === 0) break;
    children.push(...batch);
  }

  const nested = await Promise.all(children.map((c) => walkEntry(c, `${prefix}${entry.name}/`)));
  return nested.flat();
}

/**
 * Strips the common leading directory, so dropping a "dist" folder deploys
 * its contents rather than a site whose root contains a single dist/.
 */
function stripRoot(files: { path: string; file: File }[]): { path: string; file: File }[] {
  if (files.length === 0) return files;
  const first = files[0].path.split('/')[0];
  if (!files.every((f) => f.path.startsWith(first + '/'))) return files;
  return files.map((f) => ({ ...f, path: f.path.slice(first.length + 1) }));
}

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(0)} KB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MB`;
}

export function SourceDropzone({
  value,
  onChange,
}: {
  value: PickedSource | null;
  onChange: (v: PickedSource | null) => void;
}) {
  const [dragging, setDragging] = useState(false);
  const [reading, setReading] = useState(false);
  const folderInput = useRef<HTMLInputElement>(null);
  const fileInput = useRef<HTMLInputElement>(null);

  const acceptFiles = (files: { path: string; file: File }[], label: string) => {
    const kept = stripRoot(files.filter((f) => !ignored(f.path)));
    if (kept.length === 0) {
      onChange(null);
      return;
    }
    onChange({
      files: kept,
      archive: null,
      label: `${label} · ${kept.length} file${kept.length === 1 ? '' : 's'}`,
      bytes: kept.reduce((sum, f) => sum + f.file.size, 0),
    });
  };

  const onDrop = async (e: React.DragEvent) => {
    e.preventDefault();
    setDragging(false);
    setReading(true);
    try {
      const items = Array.from(e.dataTransfer.items)
        .map((i) => i.webkitGetAsEntry())
        .filter((i): i is FileSystemEntry => i !== null);

      // A single dropped archive is uploaded as-is.
      if (items.length === 1 && items[0].isFile && ARCHIVE_RE.test(items[0].name)) {
        const file = await new Promise<File>((resolve, reject) =>
          (items[0] as FileSystemFileEntry).file(resolve, reject),
        );
        onChange({ files: null, archive: file, label: file.name, bytes: file.size });
        return;
      }

      if (items.length > 0) {
        const walked = await Promise.all(items.map((i) => walkEntry(i, '')));
        const label = items.length === 1 && items[0].isDirectory ? items[0].name : 'Dropped files';
        acceptFiles(walked.flat(), label);
        return;
      }

      // Fallback for browsers without the entries API.
      const plain = Array.from(e.dataTransfer.files).map((f) => ({ path: f.name, file: f }));
      acceptFiles(plain, 'Dropped files');
    } finally {
      setReading(false);
    }
  };

  const onFolderPicked = (e: React.ChangeEvent<HTMLInputElement>) => {
    const list = Array.from(e.target.files ?? []);
    if (list.length === 0) return;
    const files = list.map((f) => ({
      // webkitRelativePath includes the chosen folder as its first segment.
      path: (f as File & { webkitRelativePath?: string }).webkitRelativePath || f.name,
      file: f,
    }));
    const root = files[0].path.split('/')[0];
    acceptFiles(files, root);
    e.target.value = '';
  };

  const onArchivePicked = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) onChange({ files: null, archive: file, label: file.name, bytes: file.size });
    e.target.value = '';
  };

  if (value) {
    const Icon = value.archive ? FileArchive : FolderOpen;
    return (
      <div className="flex items-center gap-3 rounded-[var(--radius)] border border-field-border bg-fill px-3 py-3">
        <Icon size={20} className="shrink-0 text-[var(--color-accent)]" />
        <div className="min-w-0 flex-1">
          <div className="truncate text-[length:var(--text-label)] text-text-primary">{value.label}</div>
          <div className="text-[length:var(--text-caption)] text-text-muted">{formatBytes(value.bytes)}</div>
        </div>
        <button
          type="button"
          onClick={() => onChange(null)}
          className="shrink-0 rounded p-1 text-text-muted transition-colors hover:bg-fill-hover hover:text-text-primary"
          aria-label="Remove selected source"
        >
          <X size={14} />
        </button>
      </div>
    );
  }

  return (
    <>
      <div
        role="button"
        tabIndex={0}
        onClick={() => folderInput.current?.click()}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            folderInput.current?.click();
          }
        }}
        onDragOver={(e) => {
          e.preventDefault();
          setDragging(true);
        }}
        onDragLeave={() => setDragging(false)}
        onDrop={onDrop}
        className="flex h-[120px] cursor-pointer flex-col items-center justify-center gap-2 rounded-[var(--radius)] border border-dashed text-text-muted transition-colors"
        style={{
          borderColor: dragging ? 'var(--color-accent)' : 'var(--field-border)',
          backgroundColor: dragging
            ? 'color-mix(in srgb, var(--color-accent) 8%, transparent)'
            : 'var(--fill)',
        }}
      >
        <UploadCloud size={32} className={dragging ? 'text-[var(--color-accent)]' : 'text-text-subtle'} />
        <span className="text-center text-[length:var(--text-label)]">
          {reading ? (
            'Reading files…'
          ) : (
            <>
              Drag &amp; drop your build output
              <br />
              or click to browse
            </>
          )}
        </span>
      </div>
      <button
        type="button"
        onClick={() => fileInput.current?.click()}
        className="self-start text-[length:var(--text-caption)] text-text-muted underline-offset-2 hover:text-text-primary hover:underline"
      >
        Or upload a .zip / .tar.gz archive
      </button>

      {/* webkitdirectory is non-standard but supported everywhere we target. */}
      <input
        ref={folderInput}
        type="file"
        multiple
        hidden
        onChange={onFolderPicked}
        {...({ webkitdirectory: '', directory: '' } as Record<string, string>)}
      />
      <input ref={fileInput} type="file" hidden accept=".zip,.tar.gz,.tgz" onChange={onArchivePicked} />
    </>
  );
}
