// Fix for getting sequence delays from sequence_steps table
// In ultra_optimized_broadcast_processor.go, update the query:

func (p *UltraOptimizedBroadcastProcessor) processMessages() {
	db := database.GetDB()
	
	// Get pending messages grouped by broadcast
	rows, err := db.Query(`
		SELECT 
			bm.id, bm.user_id, bm.device_id, bm.campaign_id, bm.sequence_id,
			bm.sequence_stepid, bm.recipient_phone, bm.content as message, 
			bm.media_url as image_url,
			CASE 
				WHEN bm.campaign_id IS NOT NULL THEN COALESCE(c.min_delay_seconds, 5)
				WHEN bm.sequence_stepid IS NOT NULL THEN COALESCE(ss.min_delay_seconds, 5)
				ELSE 5
			END as min_delay,
			CASE 
				WHEN bm.campaign_id IS NOT NULL THEN COALESCE(c.max_delay_seconds, 15)
				WHEN bm.sequence_stepid IS NOT NULL THEN COALESCE(ss.max_delay_seconds, 15)
				ELSE 15
			END as max_delay,
			d.status as device_status
		FROM broadcast_messages bm
		LEFT JOIN campaigns c ON bm.campaign_id = c.id
		LEFT JOIN sequence_steps ss ON bm.sequence_stepid = ss.id
		LEFT JOIN user_devices d ON bm.device_id = d.id
		WHERE bm.status = 'pending'
		AND bm.scheduled_at <= NOW()
		ORDER BY bm.campaign_id NULLS LAST, bm.sequence_id NULLS LAST, bm.created_at
		LIMIT 1000
	`)
