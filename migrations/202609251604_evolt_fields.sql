-- Rescope scans from InBody's field set to Evolt 360's (the actual scanner
-- used at the gym): add segmental fat mass alongside existing segmental lean
-- mass, and add BMR/TEE.

alter table scans
    add column if not exists fat_left_arm numeric,
    add column if not exists fat_right_arm numeric,
    add column if not exists fat_trunk numeric,
    add column if not exists fat_left_leg numeric,
    add column if not exists fat_right_leg numeric,
    add column if not exists bmr numeric,
    add column if not exists tee numeric;
