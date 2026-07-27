import { useNavigate, useParams } from 'react-router-dom';
import { CaptureReplay } from './CaptureReplay';

/*
 * The public replay a share link opens.
 *
 * Anyone with the link sees the recording — video, console, network — read-only,
 * without logging in. This is a normal product feature (sharing a bug replay),
 * so it lives in the open core, not behind a commercial gate: a self-hosted
 * install shares its own captures the same way the cloud does.
 *
 * The token in the path is the only credential. The replay loads through the
 * unauthenticated shared-captures endpoint, so nothing here reads the session.
 */
export function PublicCapturePage() {
  const { token } = useParams<{ token: string }>();
  const navigate = useNavigate();
  if (!token) return null;
  return <CaptureReplay token={token} onClose={() => navigate('/')} />;
}
