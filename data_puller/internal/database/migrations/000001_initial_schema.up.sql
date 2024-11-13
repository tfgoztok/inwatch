CREATE TYPE protocol_type AS ENUM ('snmp', 'tcp', 'modbus', 'rest');
CREATE TYPE device_status AS ENUM ('active', 'inactive', 'error');
CREATE TYPE data_type AS ENUM ('string', 'int', 'float', 'boolean');

CREATE TABLE devices (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    ip VARCHAR(255) NOT NULL,
    port INTEGER NOT NULL,
    protocol protocol_type NOT NULL,
    status device_status NOT NULL DEFAULT 'inactive',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (ip, port, protocol)
);

CREATE TABLE device_queries (
    id SERIAL PRIMARY KEY,
    device_id INTEGER NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    query_string VARCHAR(255) NOT NULL,
    parameter_name VARCHAR(255) NOT NULL,
    data_type data_type NOT NULL,
    poll_interval INTEGER NOT NULL DEFAULT 60,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (device_id, parameter_name)
);

CREATE TABLE device_configs (
    id SERIAL PRIMARY KEY,
    device_id INTEGER NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    config_key VARCHAR(255) NOT NULL,
    config_value TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (device_id, config_key)
);

-- Trigger for updating updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_devices_updated_at
    BEFORE UPDATE ON devices
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_device_queries_updated_at
    BEFORE UPDATE ON device_queries
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_device_configs_updated_at
    BEFORE UPDATE ON device_configs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Notify function for configuration changes
CREATE OR REPLACE FUNCTION notify_config_change()
RETURNS TRIGGER AS $$
DECLARE
    payload TEXT;
BEGIN
    CASE TG_TABLE_NAME
    WHEN 'devices' THEN
        payload = json_build_object(
            'table', TG_TABLE_NAME,
            'action', TG_OP,
            'device_id', COALESCE(NEW.id, OLD.id),
            'data', row_to_json(NEW)
        )::text;
    WHEN 'device_queries' THEN
        payload = json_build_object(
            'table', TG_TABLE_NAME,
            'action', TG_OP,
            'device_id', COALESCE(NEW.device_id, OLD.device_id),
            'data', row_to_json(NEW)
        )::text;
    WHEN 'device_configs' THEN
        payload = json_build_object(
            'table', TG_TABLE_NAME,
            'action', TG_OP,
            'device_id', COALESCE(NEW.device_id, OLD.device_id),
            'data', row_to_json(NEW)
        )::text;
    END CASE;

    PERFORM pg_notify('config_changes', payload);
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER notify_device_changes
    AFTER INSERT OR UPDATE OR DELETE ON devices
    FOR EACH ROW EXECUTE FUNCTION notify_config_change();

CREATE TRIGGER notify_query_changes
    AFTER INSERT OR UPDATE OR DELETE ON device_queries
    FOR EACH ROW EXECUTE FUNCTION notify_config_change();

CREATE TRIGGER notify_config_changes
    AFTER INSERT OR UPDATE OR DELETE ON device_configs
    FOR EACH ROW EXECUTE FUNCTION notify_config_change();
