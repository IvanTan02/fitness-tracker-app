-- Add Evolt 360's top-level Lean Body Mass and Body Fat Mass metrics,
-- distinct from body_fat (%) and the segmental lean/fat breakdowns.

alter table scans
    add column if not exists lean_body_mass numeric,
    add column if not exists body_fat_mass numeric;
