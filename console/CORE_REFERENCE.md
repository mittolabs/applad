# console-react core API reference (for feature ports)

You are porting ONE Flutter feature page to React. **Reuse this frozen core — never
re-implement primitives, tables, dialogs, fetching, or pagination.** Match the Flutter
source's tabs, flows, and endpoints. Path alias `@/` = `src/`.

## Control-size standard (enforced)
- **Every shadcn `<Button>` is `sm` by default** (32px, `--control-h`). Do NOT pass `size` — just `<Button>`. Only pass `size="default"|"lg"` for a deliberately larger CTA (rare). `size="icon"` is a 32px square.
- **Inputs/Selects/search fields align to `--control-h` (32px)** so controls line up in toolbars and forms. Use the `Input`/`Select`/`Textarea` core components — don't hand-roll `<input>` with a custom height (a hidden file input is the only exception).
- Toolbar chips use `<Button variant="toolbar" size="sm">` (fieldFill bg). The height token lives in `src/theme/tokens.css` (`--control-h`) — change it in one place to restyle every control.

## Rules
- Create files ONLY under `src/features/<feature>/`. Export a named component `PascalCasePage`.
- **DO NOT edit** `src/router.tsx`, `src/lib/nav.ts`, or `package.json` (the parent wires routes).
- Project-scoped pages: `const { projectId } = useParams()`. The `X-Applad-Project` header is
  ALREADY set by the shell — just call `api.get('/databases')` etc. (relative to `/v1`).
- Page shell: `<div className="flex flex-col gap-6 p-6 md:p-8">` with an `<h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">Title</h1>`.
- TypeScript strict, `noUnusedLocals`. Data is untyped JSON with Appwrite-style keys (`$id`, `$createdAt`). Type as `Record<string, unknown>` and cast at use.

## Styling (Tailwind + tokens)
Utilities: `bg-background bg-surface bg-surface-alt bg-fill bg-fill-hover bg-fill-active`,
`text-text-primary text-text-secondary text-text-muted text-text-subtle`, `border-border`,
`text-status-success|warning|danger|info|neutral`. Accent: `var(--color-accent)` (#3472A4) —
e.g. `bg-[var(--color-accent)] text-white`, focus `focus:border-[var(--color-accent)]`.
Radius: `rounded-[var(--radius)]` (8), `-10`, `-12`, `-6`, `-sm`(4). Font sizes:
`text-[length:var(--text-body)]` (13), `--text-label`(12), `--text-caption`(11),
`--text-control`(14), `--text-subhead`(15), `--text-title`(16), `--text-h1`(20), `--text-h2`(24).
Mono: `font-[family-name:var(--font-mono)]`. Icons: `lucide-react`.

## API
```ts
import { api, friendlyError } from '@/api/client'; // api = axios instance, baseURL /v1
// api.get(path, { params }) / api.post(path, data) / api.put / api.patch / api.delete(path)
```

## Data hooks (the reuse engine)
```ts
import { useResourceList } from '@/hooks/use-resource-list';
const list = useResourceList({ endpoint: '/databases', itemsKey: 'databases', scope: [projectId] });
// -> { rows, total, page, perPage, search, filters, isLoading, isFetching, error,
//      setSearch, runSearch, setPerPage, nextPage, prevPage, setFilters, refetch }

import { useTabIndex } from '@/hooks/use-tab-param';      // ?tab= synced
const [tab, setTab] = useTabIndex(['Rows','Columns']);    // [number, (i)=>void]

// TanStack Query directly for detail/nested reads; useResource for CRUD:
import { useResource } from '@/hooks/use-resource';
const { create, update, remove } = useResource('/databases'); // .mutate(...)/.isPending
```

## Components
```ts
import { Button } from '@/components/ui/button';
// variant: primary|secondary|outline|ghost|destructive|link ; size: default|sm|lg|icon ; loading; asChild
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Checkbox } from '@/components/ui/checkbox';   // onCheckedChange
import { Switch } from '@/components/ui/switch';        // onCheckedChange
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogBody, DialogFooter } from '@/components/ui/dialog';
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuCheckboxItem, DropdownMenuLabel } from '@/components/ui/dropdown-menu';
import { Popover, PopoverTrigger, PopoverContent } from '@/components/ui/popover';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'; // Radix; prefer PageTabs for page tabs
import { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider } from '@/components/ui/tooltip';
import { ScrollArea } from '@/components/ui/scroll-area';

import { PageTabs } from '@/components/page-tabs';        // {tabs:string[], selected:number, onChange:(i)=>void}
import { SearchListHeader, SearchListFooter } from '@/components/search-list';
import { StatusChip, statusVariant } from '@/components/status-chip'; // <StatusChip label="active"/> auto-maps; or variant=
import { AppBadge } from '@/components/app-badge';         // {label, icon?, color?}
import { IdText } from '@/components/id-text';             // {id}
import { CodeBlock } from '@/components/code-block';       // {code, language?}
import { EmptyState } from '@/components/empty-state';     // {icon?, title, subtitle?, actionLabel?, onAction?}
import { ErrorState } from '@/components/error-state';     // {error, onRetry?}
import { RichTextEditor } from '@/components/rich-text-editor'; // {value, onChange}
import { FormDialog, ConfirmDialog, FormField, TextField, TextAreaField, SelectField } from '@/components/form-dialog';
```

### DataTable (the main reuse win) — `@/components/data-table`
```tsx
import { DataTable, type DataTableColumn } from '@/components/data-table';
const columns: DataTableColumn[] = [
  { key: 'name', label: 'Name', flex: 2, sortable: true },
  { key: '$id', label: 'ID' },            // $id / id auto-render <IdText>
  { key: 'status', label: 'Status' },
];
<DataTable
  columns={columns}
  rows={list.rows}
  getCellValue={(row, key) => String(row[key] ?? '')}
  cellRender={(row, key) => key === 'status'
    ? <StatusChip label={String(row.status)} /> : undefined}
  rowIcon={() => Database}
  onRowClick={(row) => setSelected(row)}
  onDeleteRow={async (row) => { await api.delete(`/databases/${row.$id}`); list.refetch(); }}
  createLabel="Create database" onCreate={() => setCreating(true)}
  searchHint="Search…" searchValue={list.search} onSearchChange={list.setSearch} onSearch={list.runSearch}
  total={list.total} perPage={list.perPage} page={list.page}
  onPerPageChange={list.setPerPage} onPrev={list.prevPage} onNext={list.nextPage}
  emptyIcon={Database} emptyTitle="No databases" emptySubtitle="Create one to begin."
  loading={list.isLoading} error={list.error} onRetry={list.refetch}
/>
```

### FormDialog / ConfirmDialog
```tsx
<FormDialog open={open} onOpenChange={setOpen} title="Create X" subtitle="…"
  submitLabel="Create" loading={m.isPending} submitDisabled={!name.trim()} onSubmit={() => m.mutate()}>
  <TextField label="Name" value={name} onChange={(e)=>setName(e.target.value)} autoFocus />
  <SelectField label="Type" value={type} onChange={setType} options={[{value:'a',label:'A'}]} />
</FormDialog>

<ConfirmDialog open={!!target} onOpenChange={(o)=>!o&&setTarget(null)} title="Delete X"
  message="This cannot be undone." loading={del.isPending} onConfirm={()=>del.mutate()} />
```

### Detail-page pattern (list → detail with tabs)
Many features are: a list (DataTable), and clicking a row opens a detail view with its own PageTabs.
Hold `const [selected, setSelected] = useState<Row|null>(null)`; render the list when null, else the
detail (with a back button + `<PageTabs>`). Nested reads use `useQuery` from `@tanstack/react-query`.

### Monaco (SQL / function source)
```tsx
import Editor from '@monaco-editor/react';
<Editor height="360px" theme="vs-dark" defaultLanguage="sql" value={sql} onChange={(v)=>setSql(v??'')}
  options={{ minimap:{enabled:false}, fontSize:13 }} />
```

### Toast (success/error feedback — replaces Flutter SnackBars)
```ts
import { toast } from '@/components/toast';
toast.success('Saved'); toast.error(friendlyError(e)); toast.info('…');
```

### DeployCreateEntry (template/repo/upload wizard — Sites/Containers/Mobile/Desktop)
```tsx
import { DeployCreateEntry, type CreateEntryResult } from '@/components/deploy-create-entry';
<DeployCreateEntry open={open} onOpenChange={setOpen} category="sites" title="Create site"
  subtitle="Deploy a static site." onResult={(r: CreateEntryResult) => {
    // r.choice: 'template' | 'repository' | 'upload'; r.templateConfig / r.repoConfig
    // then POST /deploy/targets to create the target
  }} />
```

Reuse aggressively. Keep the component a THIN composition. Build must pass `tsc` strict.
