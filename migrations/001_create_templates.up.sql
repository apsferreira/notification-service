-- Create templates table
CREATE TABLE IF NOT EXISTS templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(10) NOT NULL CHECK (type IN ('email', 'sms')),
    subject_template TEXT NOT NULL DEFAULT '',
    body_template TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_templates_name ON templates(name);
CREATE INDEX idx_templates_type ON templates(type);

-- Add comments
COMMENT ON TABLE templates IS 'Notification templates for Instituto Itinerante ecosystem';
COMMENT ON COLUMN templates.type IS 'Template type: email, sms';
COMMENT ON COLUMN templates.subject_template IS 'Subject template with {{variable}} placeholders';
COMMENT ON COLUMN templates.body_template IS 'Body template with {{variable}} placeholders';
