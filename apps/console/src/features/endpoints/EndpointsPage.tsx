import { useParams } from 'react-router-dom';
import { useRoutedSelection } from '@/hooks/use-routed-selection';
import { DetailRoute } from '@/components/detail-route';
import { EndpointList } from './EndpointList';
import { EndpointBuilder } from './EndpointBuilder';

/**
 * Endpoints: a visual REST endpoint builder. Request in, blocks in the middle,
 * response out. The no-code sibling of Functions, built on the same node graph
 * as Workflows but run synchronously in-process. Separate from Workflows: its
 * own list, its own storage.
 *
 * List ⇄ Builder via a routed selection, so a refresh keeps you in the builder.
 */
export function EndpointsPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const selection = useRoutedSelection('endpoints', 'endpointId');

  if (selection.id) {
    return (
      <DetailRoute endpoint="/endpoints" id={selection.id}>
        {(endpoint, refetch) => (
          <EndpointBuilder endpoint={endpoint} onBack={selection.clear} onSaved={refetch} />
        )}
      </DetailRoute>
    );
  }

  return (
    <EndpointList
      projectId={projectId}
      onSelect={(row) => selection.select(String(row['$id'] ?? row['id'] ?? ''))}
    />
  );
}
