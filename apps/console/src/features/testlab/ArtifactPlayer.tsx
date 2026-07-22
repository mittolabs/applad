import { useEffect, useState } from 'react';
import { api } from '@/api/client';

/*
 * Plays a stored recording.
 *
 * Fetched through the API client rather than pointed at by the media element:
 * a <video src> is a plain browser request carrying neither the bearer token
 * nor the project header, so the file would come back 401.
 */
export function ArtifactPlayer({
  artifactId,
  kind,
  label,
}: {
  artifactId: string;
  kind: string;
  label?: string;
}) {
  const [src, setSrc] = useState<string | null>(null);

  useEffect(() => {
    let url: string | null = null;
    let cancelled = false;
    api
      .get(`/tests/artifacts/${artifactId}`, { responseType: 'blob' })
      .then((res) => {
        if (cancelled) return;
        url = URL.createObjectURL(res.data as Blob);
        setSrc(url);
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
      if (url) URL.revokeObjectURL(url);
    };
  }, [artifactId]);

  return (
    <div className="overflow-hidden rounded-[var(--radius)] border border-border bg-surface-alt">
      {src ? (
        kind === 'video' ? (
          <video src={src} controls preload="metadata" className="w-full bg-black" />
        ) : (
          <img src={src} alt={label ?? 'Screenshot'} className="w-full" />
        )
      ) : (
        <div className="flex h-[160px] items-center justify-center text-[length:var(--text-caption)] text-text-subtle">
          Loading recording...
        </div>
      )}
      {label && (
        <div className="truncate px-3 py-2 font-mono text-[length:var(--text-caption)] text-text-muted">
          {label}
        </div>
      )}
    </div>
  );
}
