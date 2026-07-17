import { create } from 'zustand';
import { setProject } from '@/api/client';

/* Project store — ports project_provider.dart currentProjectProvider.
 * The current project is URL-driven (/project/:projectId/...). The shell calls
 * syncProject() on every render so the X-Applad-Project header is set BEFORE
 * in-flight requests fire (mirrors AppShell._syncProject). */
interface ProjectState {
  currentProjectId: string | null;
  syncProject: (id: string | null) => void;
}

export const useProjectStore = create<ProjectState>((set, get) => ({
  currentProjectId: null,
  syncProject: (id) => {
    // Set the header synchronously (does not depend on React state commit).
    setProject(id);
    if (get().currentProjectId !== id) set({ currentProjectId: id });
  },
}));
