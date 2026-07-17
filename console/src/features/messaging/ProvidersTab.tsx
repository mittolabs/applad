import { Bell, Mail, MessageSquare, type LucideIcon } from 'lucide-react';

/* Ports _ProvidersTab — static configuration reference cards grouped by
 * category, showing each provider's required environment variables. */

interface Provider {
  icon: LucideIcon;
  category: 'Email' | 'SMS' | 'Push';
  title: string;
  subtitle: string;
  vars: string;
}

const PROVIDERS: Provider[] = [
  { icon: Mail, category: 'Email', title: 'SMTP', subtitle: 'Send emails via any SMTP server (Gmail, SendGrid, etc.)', vars: 'SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS, SMTP_FROM' },
  { icon: Mail, category: 'Email', title: 'Mailgun', subtitle: 'Send emails via the Mailgun API', vars: 'MAILGUN_API_KEY, MAILGUN_DOMAIN' },
  { icon: Mail, category: 'Email', title: 'Resend', subtitle: 'Send emails via the Resend API', vars: 'RESEND_API_KEY' },
  { icon: MessageSquare, category: 'SMS', title: 'Twilio', subtitle: 'Send SMS messages via Twilio', vars: 'TWILIO_SID, TWILIO_TOKEN, TWILIO_FROM' },
  { icon: MessageSquare, category: 'SMS', title: 'Vonage (Nexmo)', subtitle: 'Send SMS messages via the Vonage API', vars: 'VONAGE_API_KEY, VONAGE_API_SECRET, VONAGE_FROM' },
  { icon: MessageSquare, category: 'SMS', title: 'MSG91', subtitle: 'Send SMS messages via MSG91 (India)', vars: 'MSG91_AUTH_KEY, MSG91_SENDER_ID' },
  { icon: Bell, category: 'Push', title: 'Firebase Cloud Messaging (FCM)', subtitle: 'Send Android & web push notifications via FCM legacy API', vars: 'FCM_SERVER_KEY' },
  { icon: Bell, category: 'Push', title: 'Apple Push Notification Service (APNS)', subtitle: 'Send iOS push notifications via APNS HTTP/2', vars: 'APNS_KEY_ID, APNS_TEAM_ID, APNS_KEY_PATH, APNS_BUNDLE_ID' },
];

const CATEGORIES: Provider['category'][] = ['Email', 'SMS', 'Push'];

export function ProvidersTab() {
  return (
    <div className="flex flex-col gap-6 pb-8">
      {CATEGORIES.map((cat) => (
        <div key={cat} className="flex flex-col gap-2">
          <div className="text-[length:var(--text-caption)] font-semibold uppercase tracking-[0.6px] text-text-subtle">
            {cat}
          </div>
          {PROVIDERS.filter((p) => p.category === cat).map((p) => (
            <div
              key={p.title}
              className="flex items-start gap-4 rounded-[var(--radius)] border border-border bg-surface p-5"
            >
              <div className="flex h-10 w-10 items-center justify-center rounded-[var(--radius)] bg-fill">
                <p.icon size={18} className="text-text-muted" />
              </div>
              <div className="flex flex-col gap-1">
                <span className="text-[length:var(--text-control)] font-medium text-text-primary">
                  {p.title}
                </span>
                <span className="text-[length:var(--text-body)] text-text-muted">
                  {p.subtitle}
                </span>
                <span className="mt-1 font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] text-text-subtle">
                  Env vars: {p.vars}
                </span>
              </div>
            </div>
          ))}
        </div>
      ))}
    </div>
  );
}
