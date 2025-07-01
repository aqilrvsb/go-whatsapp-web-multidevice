-- Check if leads_ai table exists
SELECT EXISTS (
   SELECT FROM information_schema.tables 
   WHERE table_schema = 'public'
   AND table_name = 'leads_ai'
);

-- If not, create it
CREATE TABLE IF NOT EXISTS leads_ai (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    device_id VARCHAR(255), -- Initially NULL, assigned during campaign
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(50) NOT NULL,
    email VARCHAR(255),
    niche VARCHAR(255),
    source VARCHAR(255) DEFAULT 'ai_manual',
    status VARCHAR(50) DEFAULT 'pending', -- pending, assigned, sent, failed
    target_status VARCHAR(50) DEFAULT 'prospect', -- prospect/customer
    notes TEXT,
    assigned_at TIMESTAMP, -- When assigned to device
    sent_at TIMESTAMP, -- When message was sent
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_leads_ai_user_id ON leads_ai(user_id);
CREATE INDEX IF NOT EXISTS idx_leads_ai_device_id ON leads_ai(device_id);
CREATE INDEX IF NOT EXISTS idx_leads_ai_status ON leads_ai(status);
CREATE INDEX IF NOT EXISTS idx_leads_ai_niche ON leads_ai(niche);
CREATE INDEX IF NOT EXISTS idx_leads_ai_phone ON leads_ai(phone);

-- Also create ai_campaign_progress table
CREATE TABLE IF NOT EXISTS ai_campaign_progress (
    id SERIAL PRIMARY KEY,
    campaign_id INTEGER NOT NULL,
    device_id VARCHAR(255) NOT NULL,
    leads_sent INTEGER DEFAULT 0,
    leads_failed INTEGER DEFAULT 0,
    status VARCHAR(50) DEFAULT 'active', -- active, limit_reached, failed
    last_activity TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE,
    UNIQUE(campaign_id, device_id)
);
