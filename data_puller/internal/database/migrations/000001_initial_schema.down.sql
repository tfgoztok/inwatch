DROP TRIGGER IF EXISTS notify_config_changes ON device_configs;
DROP TRIGGER IF EXISTS notify_query_changes ON device_queries;
DROP TRIGGER IF EXISTS notify_device_changes ON devices;
DROP FUNCTION IF EXISTS notify_config_change();

DROP TRIGGER IF EXISTS update_device_configs_updated_at ON device_configs;
DROP TRIGGER IF EXISTS update_device_queries_updated_at ON device_queries;
DROP TRIGGER IF EXISTS update_devices_updated_at ON devices;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS device_configs;
DROP TABLE IF EXISTS device_queries;
DROP TABLE IF EXISTS devices;

DROP TYPE IF EXISTS data_type;
DROP TYPE IF EXISTS device_status;
DROP TYPE IF EXISTS protocol_type;