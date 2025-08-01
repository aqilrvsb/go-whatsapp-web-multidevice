-- SQL Script to Update Missing Triggers
-- Based on the data provided in the Excel export
-- Run this script directly on your PostgreSQL database

-- First, let's verify which leads need updating
SELECT COUNT(*) as "Leads that will be updated"
FROM leads
WHERE id IN (
    28574, 28570, 28563, 28539, 28523, 28522, 28507, 26516, 26249, 25822,
    25424, 25392, 25306, 24434, 24425, 24424, 24402, 24212, 24211, 24110,
    23895, 23790, 23710, 23677, 23343, 23289, 23275, 23083, 22808, 22729,
    22663, 22585, 22554, 22450, 22400, 22159, 22155, 21835, 21796, 21762,
    21718, 21433, 21360, 21333, 21190, 21184, 21111, 21018, 21017, 20983,
    20683, 20657, 20640, 20424, 20421, 20374, 20345, 20285, 20216, 20208,
    20104, 19869, 19850, 19796, 19768, 19717, 18972, 18445, 18391, 18278,
    18188, 17619, 17553, 17358, 16899, 15901, 15529, 15112, 15030, 14743,
    14655, 14327, 14170, 14109, 13957, 13956, 13927, 13838, 13815, 13733,
    13312, 13177, 13114, 12771, 12468, 12335, 12141, 12132, 11958, 11724,
    11281, 11229, 11171, 10976, 10975, 10877, 10823, 10552, 10480, 10310,
    10298, 10274, 10220, 10090, 10018, 9772, 9676, 9590, 9200, 9174,
    8566, 8550, 8492, 8402, 8390, 8216, 8108, 8090, 8088, 8068,
    8056, 7710, 7626, 7610, 7418, 7414, 7412, 7105, 6625, 6255,
    6022, 5868, 5836, 5658, 5554, 2227, 884, 796, 736, 732,
    506, 490, 436, 318, 288, 212, 202, 200, 182, 172,
    160, 154, 146, 76, 62
)
AND (trigger IS NULL OR trigger = '');

-- Update all the leads with their correct triggers
BEGIN;

-- WARMEXSTART updates
UPDATE leads SET trigger = 'WARMEXSTART', updated_at = CURRENT_TIMESTAMP
WHERE id IN (28574, 28570, 28563, 28522, 28507, 23343, 23895, 23083, 22663, 22585, 
             13114, 10552, 10480, 10310, 10274, 9676, 9590, 8492, 8402, 8390, 
             8216, 7710, 7610, 7412, 5836, 2227, 736, 732, 506, 172, 76, 62)
AND (trigger IS NULL OR trigger = '');

-- HOTEXSTART updates
UPDATE leads SET trigger = 'HOTEXSTART', updated_at = CURRENT_TIMESTAMP
WHERE id IN (28539, 28523, 24212, 24211, 23677, 21835, 21190, 21184, 20421, 20216,
             19850, 18972, 18278, 13177, 11171, 10976, 10298, 8550, 7414, 6255,
             288, 202, 200, 182, 160, 154, 146)
AND (trigger IS NULL OR trigger = '');

-- COLDASMART updates
UPDATE leads SET trigger = 'COLDASMART', updated_at = CURRENT_TIMESTAMP
WHERE id IN (26249, 25306, 24434, 24425, 24424, 24110, 22450, 21796, 21762, 18445,
             14327, 13815, 12468, 9772, 8566, 7626, 436)
AND (trigger IS NULL OR trigger = '');

-- WARMASMART updates
UPDATE leads SET trigger = 'WARMASMART', updated_at = CURRENT_TIMESTAMP
WHERE id IN (25822, 25424, 23289, 22554, 22159, 21718, 20374, 20345, 19869, 19796,
             19768, 19717, 17553, 17358, 14170, 13927, 13838, 13312, 12335, 12141,
             10975, 9174, 5658)
AND (trigger IS NULL OR trigger = '');

-- HOTASMART updates
UPDATE leads SET trigger = 'HOTASMART', updated_at = CURRENT_TIMESTAMP
WHERE id IN (26516, 24402, 22808, 22400, 20683, 20657, 20285, 20104, 19768, 18391,
             14743, 7105, 318)
AND (trigger IS NULL OR trigger = '');

-- COLDEXSTART updates
UPDATE leads SET trigger = 'COLDEXSTART', updated_at = CURRENT_TIMESTAMP
WHERE id IN (25392, 23790, 23275, 22155, 21333, 20640, 20424, 15529, 15112, 15030,
             14109, 13733, 11281, 8108, 8090, 8088, 8068, 8056, 6625, 884, 212)
AND (trigger IS NULL OR trigger = '');

-- COLDVITAC updates
UPDATE leads SET trigger = 'COLDVITAC', updated_at = CURRENT_TIMESTAMP
WHERE id IN (21433, 21360, 21018, 21017, 20983, 17619, 16899, 15901, 13957, 13956,
             11958, 10823, 6022, 5554, 796)
AND (trigger IS NULL OR trigger = '');

-- HOTVITAC updates
UPDATE leads SET trigger = 'HOTVITAC', updated_at = CURRENT_TIMESTAMP
WHERE id IN (23710, 22729, 18188, 11724, 490)
AND (trigger IS NULL OR trigger = '');

-- WARMVITAC updates
UPDATE leads SET trigger = 'WARMVITAC', updated_at = CURRENT_TIMESTAMP
WHERE id IN (20208, 11229, 10090, 10018, 7418, 5868)
AND (trigger IS NULL OR trigger = '');

-- WARM updates
UPDATE leads SET trigger = 'WARM', updated_at = CURRENT_TIMESTAMP
WHERE id IN (12771, 10877, 9200)
AND (trigger IS NULL OR trigger = '');

-- WARMEXSTART updates (additional)
UPDATE leads SET trigger = 'WARMEXSTART', updated_at = CURRENT_TIMESTAMP
WHERE id IN (21111, 12132)
AND (trigger IS NULL OR trigger = '');

-- Special case: TIDAK JUMPA
UPDATE leads SET trigger = 'TIDAK JUMPA', updated_at = CURRENT_TIMESTAMP
WHERE id = 10220
AND (trigger IS NULL OR trigger = '');

COMMIT;

-- Verify the updates
SELECT 
    trigger,
    COUNT(*) as count
FROM leads
WHERE id IN (
    28574, 28570, 28563, 28539, 28523, 28522, 28507, 26516, 26249, 25822,
    25424, 25392, 25306, 24434, 24425, 24424, 24402, 24212, 24211, 24110,
    23895, 23790, 23710, 23677, 23343, 23289, 23275, 23083, 22808, 22729,
    22663, 22585, 22554, 22450, 22400, 22159, 22155, 21835, 21796, 21762,
    21718, 21433, 21360, 21333, 21190, 21184, 21111, 21018, 21017, 20983,
    20683, 20657, 20640, 20424, 20421, 20374, 20345, 20285, 20216, 20208,
    20104, 19869, 19850, 19796, 19768, 19717, 18972, 18445, 18391, 18278,
    18188, 17619, 17553, 17358, 16899, 15901, 15529, 15112, 15030, 14743,
    14655, 14327, 14170, 14109, 13957, 13956, 13927, 13838, 13815, 13733,
    13312, 13177, 13114, 12771, 12468, 12335, 12141, 12132, 11958, 11724,
    11281, 11229, 11171, 10976, 10975, 10877, 10823, 10552, 10480, 10310,
    10298, 10274, 10220, 10090, 10018, 9772, 9676, 9590, 9200, 9174,
    8566, 8550, 8492, 8402, 8390, 8216, 8108, 8090, 8088, 8068,
    8056, 7710, 7626, 7610, 7418, 7414, 7412, 7105, 6625, 6255,
    6022, 5868, 5836, 5658, 5554, 2227, 884, 796, 736, 732,
    506, 490, 436, 318, 288, 212, 202, 200, 182, 172,
    160, 154, 146, 76, 62
)
GROUP BY trigger
ORDER BY count DESC;

-- Show sample of updated leads
SELECT id, name, phone, niche, trigger, updated_at
FROM leads
WHERE id IN (
    28574, 28570, 28563, 28539, 28523
)
ORDER BY id;
