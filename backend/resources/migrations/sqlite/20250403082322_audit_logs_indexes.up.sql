CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
CREATE INDEX idx_audit_logs_event ON audit_logs(event);
CREATE INDEX idx_audit_logs_ip_address ON audit_logs(ip_address);
CREATE INDEX idx_audit_logs_user_agent ON audit_logs(user_agent);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_country ON audit_logs(country);
CREATE INDEX idx_audit_logs_city ON audit_logs(city);
CREATE INDEX idx_audit_logs_client_name ON audit_logs((json_extract(data, '$.clientName')));
