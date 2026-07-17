import {
  Atom,
  FileCode,
  Flame,
  Hexagon,
  Layers,
  Rocket,
  Smartphone,
  Triangle,
  type LucideIcon,
} from 'lucide-react';

/* Framework metadata — ports the _frameworks table in sites_page.dart. */

export interface Framework {
  id: string;
  label: string;
  icon: LucideIcon;
  buildCommand: string;
  outputDir: string;
  installCommand: string;
}

export const FRAMEWORKS: Framework[] = [
  { id: 'nextjs', label: 'Next.js', icon: Hexagon, buildCommand: 'npm run build', outputDir: '.next', installCommand: 'npm install' },
  { id: 'sveltekit', label: 'SvelteKit', icon: Flame, buildCommand: 'npm run build', outputDir: 'build', installCommand: 'npm install' },
  { id: 'nuxt', label: 'Nuxt', icon: Triangle, buildCommand: 'npm run build', outputDir: '.output/public', installCommand: 'npm install' },
  { id: 'astro', label: 'Astro', icon: Rocket, buildCommand: 'npm run build', outputDir: 'dist', installCommand: 'npm install' },
  { id: 'react', label: 'React', icon: Atom, buildCommand: 'npm run build', outputDir: 'build', installCommand: 'npm install' },
  { id: 'vue', label: 'Vue', icon: Layers, buildCommand: 'npm run build', outputDir: 'dist', installCommand: 'npm install' },
  { id: 'flutter_web', label: 'Flutter Web', icon: Smartphone, buildCommand: 'flutter build web --release', outputDir: 'build/web', installCommand: 'flutter pub get' },
  { id: 'static', label: 'Static', icon: FileCode, buildCommand: '', outputDir: 'public', installCommand: '' },
];

export function frameworkById(id: string): Framework {
  return FRAMEWORKS.find((f) => f.id === id) ?? FRAMEWORKS[FRAMEWORKS.length - 1];
}
