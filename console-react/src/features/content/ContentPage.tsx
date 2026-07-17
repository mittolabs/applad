import { useState } from 'react';
import { useParams } from 'react-router-dom';
import type { ContentRow } from './shared';
import { TypesView } from './TypesView';
import { EntriesView } from './EntriesView';
import { EntryEditor } from './EntryEditor';

/* Ports console/lib/features/content/content_page.dart — a headless CMS with
 * three screens driven by selection state: Types → Entries → Entry editor. */
export function ContentPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const [selectedType, setSelectedType] = useState<ContentRow | null>(null);
  const [selectedEntry, setSelectedEntry] = useState<ContentRow | null>(null);
  const [creatingEntry, setCreatingEntry] = useState(false);

  const inEntryEditor = selectedEntry !== null || creatingEntry;

  const selectType = (t: ContentRow) => {
    setSelectedType(t);
    setSelectedEntry(null);
    setCreatingEntry(false);
  };
  const backFromEntries = () => setSelectedType(null);
  const backFromEditor = () => {
    setSelectedEntry(null);
    setCreatingEntry(false);
  };

  if (!selectedType) {
    return <TypesView projectId={projectId} onSelectType={selectType} />;
  }

  if (inEntryEditor) {
    return (
      <EntryEditor
        projectId={projectId}
        type={selectedType}
        entry={selectedEntry}
        onBack={backFromEditor}
        onSaved={backFromEditor}
      />
    );
  }

  return (
    <EntriesView
      projectId={projectId}
      type={selectedType}
      onBack={backFromEntries}
      onSelectEntry={setSelectedEntry}
      onNewEntry={() => setCreatingEntry(true)}
    />
  );
}
