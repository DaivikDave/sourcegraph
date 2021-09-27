BEGIN;

ALTER TABLE lsif_data_definitions ADD COLUMN kind TEXT;
ALTER TABLE lsif_data_references ADD COLUMN kind TEXT;

COMMENT ON COLUMN lsif_data_definitions.kind IS 'The moniker kind.';
COMMENT ON COLUMN lsif_data_references.kind IS 'The moniker kind.';

ALTER TABLE lsif_data_references DROP CONSTRAINT lsif_data_references_pkey;
ALTER TABLE lsif_data_definitions DROP CONSTRAINT lsif_data_definitions_pkey;

COMMIT;
