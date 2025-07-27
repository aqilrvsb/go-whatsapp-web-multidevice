			// Create broadcast message WITHOUT SequenceStepID to avoid UUID errors
			msg := domainBroadcast.BroadcastMessage{
				UserID:         lead.UserID,
				DeviceID:       lead.DeviceID,
				SequenceID:     &currentSequenceID,
				// Don't set SequenceStepID - it's causing UUID errors
				RecipientPhone: lead.Phone,
				RecipientName:  lead.Name,
				Message:        step.Content,
				Content:        step.Content,
				Type:           step.MessageType,
				MinDelay:       step.MinDelay,
				MaxDelay:       step.MaxDelay,
				ScheduledAt:    scheduledAt,
				Status:         "pending",
			}

			// Handle media URL
			if step.MediaURL.Valid && step.MediaURL.String != "" {
				msg.MediaURL = step.MediaURL.String
				msg.ImageURL = step.MediaURL.String
			}

			// Debug log before queueing
			logrus.Debugf("Queueing message - UserID: '%s', DeviceID: '%s', SequenceID: '%s'", 
				msg.UserID, msg.DeviceID, *msg.SequenceID)

			allMessages = append(allMessages, msg)