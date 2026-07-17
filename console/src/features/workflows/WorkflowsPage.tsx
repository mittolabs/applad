import { useState } from 'react';
import { useParams } from 'react-router-dom';
import type { Row } from '@/components/data-table';
import { WorkflowList } from './WorkflowList';
import { WorkflowBuilder } from './WorkflowBuilder';

/**
 * Workflows — n8n-style visual DAG builder.
 * Ports console/lib/features/workflows/workflows_page.dart to React + React Flow.
 * List (DataTable) ⇄ Builder (React Flow canvas) via a selection pattern.
 */
export function WorkflowsPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const [editing, setEditing] = useState<Row | null>(null);

  if (editing) {
    return (
      <WorkflowBuilder
        workflow={editing}
        onBack={() => setEditing(null)}
        onSaved={() => {
          /* list refetches on return via its own hook scope */
        }}
      />
    );
  }

  return <WorkflowList projectId={projectId} onSelect={setEditing} />;
}
