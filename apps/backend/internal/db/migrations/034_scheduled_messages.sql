-- Scheduled message delivery. A recorded message given a future scheduledAt is
-- stored with status 'scheduled' and delivered by the per-minute cron sweep
-- once that time has passed (see SweepScheduledMessages).
--
-- The sweep asks for rows that are still 'scheduled' and already due, once a
-- minute, so index exactly those rows by their scheduled time. A partial index
-- stays tiny: once a message is delivered it leaves 'scheduled' and drops out.
CREATE INDEX IF NOT EXISTS idx_messages_scheduled
    ON messages (scheduled_at)
    WHERE status = 'scheduled';
