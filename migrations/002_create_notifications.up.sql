-- Create notifications table
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(10) NOT NULL CHECK (type IN ('email', 'sms')),
    recipient VARCHAR(500) NOT NULL,
    subject VARCHAR(500) NOT NULL DEFAULT '',
    body TEXT NOT NULL DEFAULT '',
    template_id UUID REFERENCES templates(id) ON DELETE SET NULL,
    variables JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'failed', 'retrying')),
    attempts INTEGER NOT NULL DEFAULT 0,
    error TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    sent_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes for better performance
CREATE INDEX idx_notifications_type ON notifications(type);
CREATE INDEX idx_notifications_status ON notifications(status);
CREATE INDEX idx_notifications_recipient ON notifications(recipient);
CREATE INDEX idx_notifications_template_id ON notifications(template_id);
CREATE INDEX idx_notifications_created_at ON notifications(created_at);
CREATE INDEX idx_notifications_pending_retries ON notifications(status, attempts) WHERE status IN ('pending', 'retrying') AND attempts < 3;

-- Create GIN index for JSONB variables search
CREATE INDEX idx_notifications_variables ON notifications USING GIN(variables);

-- Add comments
COMMENT ON TABLE notifications IS 'Notification log for Instituto Itinerante ecosystem';
COMMENT ON COLUMN notifications.type IS 'Notification type: email, sms';
COMMENT ON COLUMN notifications.status IS 'Notification status: pending, sent, failed, retrying';
COMMENT ON COLUMN notifications.variables IS 'Template variables used for rendering';
COMMENT ON COLUMN notifications.attempts IS 'Number of send attempts (max 3)';
