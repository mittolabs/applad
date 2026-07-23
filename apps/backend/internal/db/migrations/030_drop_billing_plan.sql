-- Remove the organizations.billing_plan column.
--
-- A plan is a commercial concept, not a property of the open core: a self-hosted
-- install has no plans, and the hosted product derives an org's plan from its
-- subscription in the commercial billing layer. The column was always just
-- 'free' here and only leaked the commercial model into the BSD-3 schema.
--
-- Idempotent (IF EXISTS): fresh installs never had it (001_init no longer
-- creates it), existing installs drop it here.
ALTER TABLE organizations DROP COLUMN IF EXISTS billing_plan;
