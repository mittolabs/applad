-- ---------------------------------------------------------------------------
-- Migration 045: Remove Plan.
--
-- Plan was Linear inside a backend console: items, milestones, acceptance
-- criteria, comments and a priority matrix. Nothing else in the platform read
-- from it, no SDK exposed it, and it sat first in the rail — so the first thing
-- a new user saw on a backend platform was a project-management tool. A team
-- adopting Applad already plans somewhere else, and will not move planning to
-- their backend vendor.
-- ---------------------------------------------------------------------------

-- CASCADE takes criteria, comments, links, activity and the priority tables
-- with their parents, so the order of these statements does not matter.
DROP TABLE IF EXISTS plan_priority_answers   CASCADE;
DROP TABLE IF EXISTS plan_priority_options   CASCADE;
DROP TABLE IF EXISTS plan_priority_questions CASCADE;
DROP TABLE IF EXISTS plan_priority_bands     CASCADE;
DROP TABLE IF EXISTS plan_priority_matrix    CASCADE;
DROP TABLE IF EXISTS plan_activity           CASCADE;
DROP TABLE IF EXISTS plan_comments           CASCADE;
DROP TABLE IF EXISTS plan_criteria           CASCADE;
DROP TABLE IF EXISTS plan_item_links         CASCADE;
DROP TABLE IF EXISTS plan_items              CASCADE;
DROP TABLE IF EXISTS plan_milestones         CASCADE;
