import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Check, Copy, Download, File as FileIcon } from 'lucide-react';
import { api } from '@/api/client';
import { Button } from '@/components/ui/button';
import { IdText } from '@/components/id-text';
import { ConfirmDialog } from '@/components/form-dialog';
import { ErrorState } from '@/components/error-state';
import { fileViewUrl, fmtBytes } from './helpers';

/* Ports storage_page.dart _FileDetailView — file preview, metadata, URL,
 * download link and delete card. Reads the bucket's file list and finds the
 * selected file (there is no single-file GET endpoint). */

export function FileDetailView({
  bucketId,
  fileId,
  onBack,
}: {
  bucketId: string;
  fileId: string;
  onBack: () => void;
}) {
  const qc = useQueryClient();
  const [confirming, setConfirming] = useState(false);
  const [copied, setCopied] = useState(false);

  const filesKey = ['storage-files', bucketId];
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: filesKey,
    queryFn: async () => {
      const res = await api.get(`/storage/buckets/${bucketId}/files`, {
        params: { limit: 100 },
      });
      return res.data as Record<string, unknown>;
    },
  });

  const files = (data?.['files'] as Record<string, unknown>[] | undefined) ?? [];
  const file = files.find((f) => f['$id'] === fileId) ?? {};

  const name = (file['name'] as string) ?? fileId;
  const mime = (file['mimeType'] as string) ?? '';
  const size = (file['sizeOriginal'] as number) ?? 0;
  const created = file['createdAt'] ?? file['$createdAt'];
  const updated = file['updatedAt'] ?? file['$updatedAt'];
  const url = fileViewUrl(bucketId, fileId);

  const del = useMutation({
    mutationFn: () => api.delete(`/storage/buckets/${bucketId}/files/${fileId}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: filesKey });
      onBack();
    },
  });

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      /* clipboard unavailable */
    }
  };

  return (
    <div className="flex flex-col gap-6 p-6 md:p-8">
      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={onBack}
          className="text-text-muted transition-colors hover:text-text-primary"
          aria-label="Back"
        >
          <ArrowLeft size={20} />
        </button>
        <h1 className="min-w-0 truncate text-[length:var(--text-h1)] font-semibold text-text-primary">
          {name}
        </h1>
        <div className="flex items-center gap-1.5">
          <FileIcon size={13} className="text-text-subtle" />
          <IdText id={fileId} />
        </div>
      </div>

      {error ? (
        <ErrorState error={error} onRetry={() => void refetch()} />
      ) : (
        <>
          {/* Info card + preview */}
          <div className="rounded-[var(--radius)] border border-border bg-surface p-6">
            <div className="flex flex-col gap-6 md:flex-row md:items-start">
              <div className="flex h-40 w-full items-center justify-center overflow-hidden rounded-[var(--radius)] border border-border bg-field-fill md:w-52">
                {mime.startsWith('image/') ? (
                  <img
                    src={url}
                    alt={name}
                    className="max-h-full max-w-full object-contain"
                    onError={(e) => {
                      (e.currentTarget as HTMLImageElement).style.display = 'none';
                    }}
                  />
                ) : (
                  <FileIcon size={48} className="text-text-subtle" />
                )}
              </div>

              <div className="min-w-0 flex-1">
                <MetaRow label="Filename" value={name} />
                <MetaRow label="MIME type" value={mime || '—'} />
                <MetaRow label="Size" value={fmtBytes(size)} />
                {created != null && <MetaRow label="Created" value={String(created)} />}
                {updated != null && <MetaRow label="Last updated" value={String(updated)} />}

                <div className="mt-4 text-[length:var(--text-label)] text-text-muted">
                  File URL
                </div>
                <div className="mt-1 flex items-center gap-2 rounded-[var(--radius-6)] border border-field-border bg-field-fill px-2.5 py-2">
                  <span className="min-w-0 flex-1 truncate font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] text-text-muted">
                    {url}
                  </span>
                  <button
                    type="button"
                    onClick={copy}
                    className="text-text-subtle transition-colors hover:text-text-primary"
                    aria-label="Copy URL"
                  >
                    {copied ? (
                      <Check size={14} className="text-status-success" />
                    ) : (
                      <Copy size={14} />
                    )}
                  </button>
                </div>

                <Button variant="outline" size="sm" className="mt-3" asChild>
                  <a href={url} target="_blank" rel="noreferrer" download>
                    <Download size={14} />
                    Download
                  </a>
                </Button>
              </div>
            </div>
          </div>

          {/* Permissions card (static, matches Flutter) */}
          <div className="rounded-[var(--radius)] border border-border bg-surface p-5">
            <div className="text-[length:var(--text-subhead)] font-semibold text-text-primary">
              Permissions
            </div>
            <div className="mt-2 text-[length:var(--text-body)] text-text-subtle">
              Assign read or write permissions at the bucket level or file level.
            </div>
          </div>

          {/* Delete card */}
          <div className="flex items-center gap-4 rounded-[var(--radius)] border border-[color-mix(in_srgb,var(--color-danger)_30%,var(--border))] bg-surface p-5">
            <div className="flex-1">
              <div className="text-[length:var(--text-control)] font-medium text-[var(--color-danger)]">
                Delete file
              </div>
              <div className="mt-1 text-[length:var(--text-body)] text-text-subtle">
                The file will be permanently deleted. This action is irreversible.
              </div>
            </div>
            <Button
              variant="outline"
              className="border-[var(--color-danger)] text-[var(--color-danger)] hover:bg-[color-mix(in_srgb,var(--color-danger)_12%,transparent)]"
              disabled={isLoading}
              onClick={() => setConfirming(true)}
            >
              Delete
            </Button>
          </div>
        </>
      )}

      <ConfirmDialog
        open={confirming}
        onOpenChange={setConfirming}
        title="Delete file"
        message="The file will be permanently deleted. This action is irreversible."
        loading={del.isPending}
        onConfirm={() => del.mutate()}
      />
    </div>
  );
}

function MetaRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="mb-1.5 flex items-start gap-3">
      <span className="w-28 shrink-0 text-[length:var(--text-body)] text-text-muted">
        {label}
      </span>
      <span className="min-w-0 flex-1 break-words text-[length:var(--text-body)] text-text-primary">
        {value}
      </span>
    </div>
  );
}
