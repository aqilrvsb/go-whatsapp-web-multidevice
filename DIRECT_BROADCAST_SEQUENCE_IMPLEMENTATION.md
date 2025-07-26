### Technical Details:

#### Message Creation Flow:
```
1. Lead has trigger "COLDEXAMA"
2. System finds COLD sequence with entry point "COLDEXAMA" (MUST be is_active = true)
3. Creates 5 messages for COLD steps with calculated scheduled_at times
4. Finds COLD step 5 has next_trigger = "WARMEXAMA"
5. Looks for sequence with trigger = "WARMEXAMA" (does NOT check is_active)
6. Creates 4 messages for WARM steps
7. Finds WARM step 4 has next_trigger = "HOTEXAMA"
8. Creates 2 messages for HOT steps
9. Total: 11 messages created in one transaction
```

#### Important Behavior:
- **Initial enrollment**: Only happens if the starting sequence is `is_active = true`
- **Following links**: Once enrolled, follows ALL `next_trigger` links regardless of their active status
- **Example**: If COLD is active but WARM is inactive, a lead with trigger "COLDEXAMA" will still get ALL messages (COLD + WARM + HOT)
- **Example**: If WARM is inactive, a lead with trigger "WARMEXAMA" will NOT get enrolled at all