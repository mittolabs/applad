import {
  siNextdotjs,
  siSvelte,
  siNuxt,
  siAstro,
  siReact,
  siVuedotjs,
  siFlutter,
  siHtml5,
} from 'simple-icons';

/*
 * Real framework marks rather than lookalike glyphs. Rendered monochrome so a
 * grid of them stays calm, and filled with the brand colour on hover or when
 * selected.
 */

interface Brand {
  path: string;
  /** Brand colour, adjusted where the official one is invisible on dark. */
  color: string;
}

const BRANDS: Record<string, Brand> = {
  nextjs: { path: siNextdotjs.path, color: '#FFFFFF' }, // official black reads as a hole on dark
  sveltekit: { path: siSvelte.path, color: `#${siSvelte.hex}` },
  nuxt: { path: siNuxt.path, color: `#${siNuxt.hex}` },
  astro: { path: siAstro.path, color: `#${siAstro.hex}` },
  react: { path: siReact.path, color: `#${siReact.hex}` },
  vue: { path: siVuedotjs.path, color: `#${siVuedotjs.hex}` },
  flutter_web: { path: siFlutter.path, color: '#47C5FB' }, // the light half of the mark
  static: { path: siHtml5.path, color: `#${siHtml5.hex}` },
};

export function brandColor(frameworkId: string): string {
  return BRANDS[frameworkId]?.color ?? 'currentColor';
}

export function FrameworkLogo({
  framework,
  size = 24,
  active = false,
  className,
}: {
  framework: string;
  size?: number;
  /** Paint the brand colour without waiting for hover (selected rows, detail headers). */
  active?: boolean;
  className?: string;
}) {
  const brand = BRANDS[framework] ?? BRANDS.static;
  return (
    <svg
      role="img"
      aria-hidden="true"
      width={size}
      height={size}
      viewBox="0 0 24 24"
      // The brand colour rides on a custom property so a parent marked `group`
      // can light it up on hover without this component tracking hover itself.
      className={`transition-colors group-hover:text-[var(--brand)] ${className ?? ''}`}
      style={
        {
          '--brand': brand.color,
          ...(active ? { color: 'var(--brand)' } : null),
        } as React.CSSProperties
      }
    >
      <path d={brand.path} fill="currentColor" />
    </svg>
  );
}
