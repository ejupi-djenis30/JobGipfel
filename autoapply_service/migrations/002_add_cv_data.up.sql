-- Migration: Add CV data to applications
-- Store the CV used for each application as base64

ALTER TABLE applications ADD COLUMN IF NOT EXISTS cv_data TEXT;
ALTER TABLE applications ADD COLUMN IF NOT EXISTS cv_style TEXT;
ALTER TABLE applications ADD COLUMN IF NOT EXISTS cv_color_scheme TEXT;

COMMENT ON COLUMN applications.cv_data IS 'Base64 encoded PDF of the CV used for this application';
COMMENT ON COLUMN applications.cv_style IS 'CV style used (modern, minimalist, classic, creative)';
COMMENT ON COLUMN applications.cv_color_scheme IS 'Color scheme used for the CV';
