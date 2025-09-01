# WhatsApp Multi-Device Campaign & Sequence System - Complete Technical Summary

## Table of Contents
1. [System Overview](#system-overview)
2. [Campaign System](#campaign-system)
3. [Sequence System](#sequence-system)
4. [Broadcast Message Queue](#broadcast-message-queue)
5. [Processing Flow](#processing-flow)
6. [Key Features](#key-features)
7. [Implementation Guide](#implementation-guide)

---

## System Overview

This WhatsApp broadcast system supports two main types of automated messaging:

1. **Campaigns**: One-time mass broadcasts to targeted leads
2. **Sequences**: Multi-day drip campaigns with automated follow-ups

Both systems utilize a centralized **broadcast_messages** queue table for message delivery, ensuring consistent processing and preventing duplicates.

---

## Campaign System

### 1. Data Model (`models/campaign.go`)

```go
type Campaign struct {
    ID              int       // Auto-increment primary key
    UserID          string    // Owner user ID
    DeviceID        string    // DEPRECATED - campaigns now use all user devices
    Title           string    // Campaign name
    Niche           string    // Target niche/category (e.g., "EXSTART", "ITADRESS")
    TargetStatus    string    // Target lead status: "prospect", "customer", "all"
    Message         string    // Message content (supports Spintax)
    ImageURL        string    // Optional image URL