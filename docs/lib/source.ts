import { docs } from '@/.source';
import { loader } from 'fumadocs-core/source';

// baseUrl '/' — docs are served at the root of docs.applad.io.
export const source = loader({
  baseUrl: '/',
  source: docs.toFumadocsSource(),
});
