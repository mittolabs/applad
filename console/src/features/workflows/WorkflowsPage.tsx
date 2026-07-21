import { useParams } from 'react-router-dom';
import { useRoutedSelection } from '@/hooks/use-routed-selection';
import { DetailRoute } from '@/components/detail-route';
import { WorkflowList } from './WorkflowList';
import { WorkflowBuilder } from './WorkflowBuilder';

/**
 * Workflows — n8n-style visual DAG builder.
 * Ports console/lib/features/workflows/workflows_page.dart to React + React Flow.
 * List (DataTable) ⇄ Builder (React Flow canvas) via a selection pattern.
 */
export function WorkflowsPage() {
  const { projectId } = useParams<{ projectId: string }>();
  // The workflow being edited is part of the address, so a refresh does not
  // throw somebody out of a canvas they were working in.
  const selection = useRoutedSelection('workflows', 'workflowId');

  if (selection.id) {
    return (
      <DetailRoute endpoint="/workflows" id={selection.id}>
        {(workflow) => (
          <WorkflowBuilder
            workflow={workflow}
            onBack={selection.clear}
            onSaved={() => {
              /* list refetches on return via its own hook scope */
            }}
          />
        )}
      </DetailRoute>
    );
  }

  return (
    <WorkflowList
      projectId={projectId}
      onSelect={(row) => selection.select(String(row['$id'] ?? row['id'] ?? ''))}
    />
  );
}
