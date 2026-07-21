/*
 * Minimal tar + gzip writer.
 *
 * The deploy source endpoint takes a gzipped tar, which is what the build
 * worker extracts. Browsers can't produce one natively, so we write the tar
 * ourselves (it's a simple 512-byte-header format) and gzip it with the
 * platform CompressionStream. No dependency needed.
 */

export interface TarEntry {
  /** Path inside the archive, relative and slash-separated. */
  path: string;
  data: Uint8Array;
}

const BLOCK = 512;

function writeString(block: Uint8Array, offset: number, value: string, length: number) {
  const bytes = new TextEncoder().encode(value);
  block.set(bytes.subarray(0, length - 1), offset);
}

function writeOctal(block: Uint8Array, offset: number, value: number, length: number) {
  // Classic tar numeric fields: zero-padded octal, NUL-terminated.
  const text = value.toString(8).padStart(length - 1, '0');
  writeString(block, offset, text, length);
}

function header(path: string, size: number): Uint8Array {
  const block = new Uint8Array(BLOCK);

  // Paths longer than 100 bytes use the prefix field (ustar), splitting on a
  // separator. Without this, deep build outputs would be silently truncated.
  let name = path;
  let prefix = '';
  if (new TextEncoder().encode(path).length > 100) {
    const cut = path.lastIndexOf('/', path.length - 100);
    if (cut > 0) {
      prefix = path.slice(0, cut);
      name = path.slice(cut + 1);
    }
  }

  writeString(block, 0, name, 100);
  writeOctal(block, 100, 0o644, 8); // mode
  writeOctal(block, 108, 0, 8); // uid
  writeOctal(block, 116, 0, 8); // gid
  writeOctal(block, 124, size, 12);
  writeOctal(block, 136, Math.floor(Date.now() / 1000), 12); // mtime
  block[156] = 0x30; // typeflag '0' — regular file
  writeString(block, 257, 'ustar', 6);
  block[263] = 0x30; // version "00"
  block[264] = 0x30;
  writeString(block, 345, prefix, 155);

  // Checksum is computed with the checksum field itself read as spaces.
  block.fill(0x20, 148, 156);
  let sum = 0;
  for (const byte of block) sum += byte;
  writeOctal(block, 148, sum, 8);
  block[155] = 0x20;

  return block;
}

/** Builds an uncompressed tar archive from the given entries. */
export function buildTar(entries: TarEntry[]): Uint8Array {
  const chunks: Uint8Array[] = [];
  let total = 0;

  for (const entry of entries) {
    const head = header(entry.path, entry.data.length);
    chunks.push(head, entry.data);
    total += head.length + entry.data.length;

    // File contents are padded out to a 512-byte boundary.
    const padding = (BLOCK - (entry.data.length % BLOCK)) % BLOCK;
    if (padding > 0) {
      const pad = new Uint8Array(padding);
      chunks.push(pad);
      total += padding;
    }
  }

  // Two zero blocks mark end of archive.
  const trailer = new Uint8Array(BLOCK * 2);
  chunks.push(trailer);
  total += trailer.length;

  const out = new Uint8Array(total);
  let offset = 0;
  for (const chunk of chunks) {
    out.set(chunk, offset);
    offset += chunk.length;
  }
  return out;
}

/** Builds a gzipped tar archive, ready to POST to the source endpoint. */
export async function buildTarGz(entries: TarEntry[]): Promise<Blob> {
  const tar = buildTar(entries);
  const gzip = new CompressionStream('gzip');
  const stream = new Blob([tar as BlobPart]).stream().pipeThrough(gzip);
  return new Response(stream).blob();
}
