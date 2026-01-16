-- Rollback: Remove CV data columns

ALTER TABLE applications DROP COLUMN IF EXISTS cv_data;
ALTER TABLE applications DROP COLUMN IF EXISTS cv_style;
ALTER TABLE applications DROP COLUMN IF EXISTS cv_color_scheme;
