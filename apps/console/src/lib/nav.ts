import {
  Activity,
  BarChart3,
  Box,
  Clock,
  Code2,
  Container,
  Database,
  Flag,
  Filter,
  FolderOpen,
  Globe,
  KeyRound,
  DownloadCloud,
  type LucideIcon,
  LayoutDashboard,
  LineChart,
  Mail,
  Monitor,
  Radio,
  Rocket,
  Settings,
  Smartphone,
  Users,
  Waypoints,
  Workflow,
  Zap,
} from 'lucide-react';

/* Nav model — ports shell.dart _buildGroups(). `route` is relative to
 * /project/:projectId/. Groups with no children navigate directly; groups with
 * children expand the detail panel and auto-navigate to the first child. */
export interface NavChild {
  label: string;
  route: string;
  icon: LucideIcon;
}
export interface NavGroup {
  id: string;
  label: string;
  icon: LucideIcon;
  route?: string; // direct-nav groups (no children)
  children?: NavChild[];
  pinBottom?: boolean;
}

export const navGroups: NavGroup[] = [
  { id: 'overview', label: 'Overview', icon: BarChart3, route: 'overview' },
  {
    id: 'build',
    label: 'Build',
    icon: Box,
    children: [
      { label: 'Auth', route: 'auth', icon: Users },
      { label: 'Databases', route: 'databases', icon: Database },
      { label: 'Functions', route: 'functions', icon: Code2 },
      { label: 'Endpoints', route: 'endpoints', icon: Waypoints },
      { label: 'Storage', route: 'storage', icon: FolderOpen },
      { label: 'Messaging', route: 'messaging', icon: Mail },
      { label: 'Realtime', route: 'realtime', icon: Radio },
      { label: 'Feature Flags', route: 'flags', icon: Flag },
    ],
  },
  {
    // Every child here is a deploy target — /deploy/targets filtered by type —
    // so "Platforms" named the container rather than the act, and collided
    // with what Platforms means in Appwrite, where it registers client apps.
    id: 'deploy',
    label: 'Deploy',
    icon: Rocket,
    children: [
      { label: 'Sites', route: 'sites', icon: Globe },
      { label: 'Containers', route: 'containers', icon: Container },
      { label: 'Mobile', route: 'mobile', icon: Smartphone },
      { label: 'Desktop', route: 'desktop', icon: Monitor },
    ],
  },
  {
    id: 'automate',
    label: 'Automate',
    icon: Zap,
    children: [{ label: 'Workflows', route: 'workflows', icon: Workflow }],
  },
  {
    // Errors, logs, replays and alerts left with Bugslad. What remains is what
    // the platform measures about itself, plus the product analytics it was
    // already collecting and never showed.
    id: 'analytics',
    label: 'Analytics',
    icon: LineChart,
    children: [
      { label: 'Overview', route: 'analytics', icon: LayoutDashboard },
      { label: 'Events', route: 'events', icon: BarChart3 },
      { label: 'Funnels', route: 'funnels', icon: Filter },
      { label: 'Uptime', route: 'uptime', icon: Activity },
      { label: 'Crons', route: 'crons', icon: Clock },
    ],
  },
  {
    id: 'settings',
    label: 'Settings',
    icon: Settings,
    pinBottom: true,
    children: [
      { label: 'General', route: 'settings', icon: Settings },
      { label: 'Team', route: 'auth?tab=teams', icon: Users },
      { label: 'Vault', route: 'vault', icon: KeyRound },
      { label: 'Migrations', route: 'migrations', icon: DownloadCloud },
    ],
  },
];

/** Keyboard chord shortcuts: press "g" then the key to jump to a section.
 * Keyed by route segment. Shown in the ⌘K palette and handled by the shell. */
export const navShortcuts: Record<string, string> = {
  overview: 'o',
  auth: 'a',
  databases: 'd',
  functions: 'f',
  endpoints: 'e',
  storage: 's',
  messaging: 'm',
  workflows: 'w',
  settings: ',',
};

/** Reverse lookup: chord key → route segment (for the shell listener). */
export const shortcutToRoute: Record<string, string> = Object.fromEntries(
  Object.entries(navShortcuts).map(([route, key]) => [key, route]),
);

/** Map the first path segment after /project/:id/ to its owning group id. */
export const routeToGroup: Record<string, string> = (() => {
  const map: Record<string, string> = {};
  for (const g of navGroups) {
    if (g.route && !(g.route in map)) map[g.route] = g.id;
    for (const c of g.children ?? []) {
      const seg = c.route.split('?')[0].split('/')[0];
      // Don't overwrite a segment already owned by an earlier group — e.g. the
      // Settings "Team" item routes to `auth?tab=teams`, but `auth` belongs to Build.
      if (!(seg in map)) map[seg] = g.id;
    }
  }
  // extra segments that live under a group but aren't nav children
  map['get-started'] = 'overview';
  map['platforms'] = 'deploy';
  map['health'] = 'analytics';
  map['environments'] = 'settings';
  return map;
})();
