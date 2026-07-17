import {
  KeyRound,
  Link as LinkIcon,
  type LucideIcon,
  Mail,
  Phone,
  Shield,
  UserPlus,
  UserX,
} from 'lucide-react';

/* Ports the static `_authMethods` and `_oauthProviders` tables from
 * auth_page.dart — drives the Settings tab (auth methods + OAuth providers). */

export interface AuthMethod {
  id: string;
  label: string;
  description: string;
  icon: LucideIcon;
  defaultOn?: boolean;
}

export const AUTH_METHODS: AuthMethod[] = [
  { id: 'email', label: 'Email / Password', description: 'Sign in with email and password', icon: Mail, defaultOn: true },
  { id: 'phone', label: 'Phone', description: 'Sign in with phone number + OTP', icon: Phone },
  { id: 'magic', label: 'Magic URL', description: 'Passwordless email link sign-in', icon: LinkIcon },
  { id: 'emailOtp', label: 'Email OTP', description: 'Sign in with one-time code via email', icon: KeyRound },
  { id: 'anonymous', label: 'Anonymous', description: 'Sessions without credentials', icon: UserX, defaultOn: true },
  { id: 'teamInvites', label: 'Team Invites', description: 'Accept team invitations to sign up', icon: UserPlus, defaultOn: true },
  { id: 'jwt', label: 'JWT', description: 'Accept externally issued JWT tokens', icon: Shield },
];

export type ProviderFieldType = 'text' | 'secret' | 'multiline';

export interface ProviderField {
  key: string;
  label: string;
  hint: string;
  type?: ProviderFieldType;
}

export interface OAuthProvider {
  id: string;
  name: string;
  color: string;
  letter: string;
  fields: ProviderField[];
  setupNote?: string;
}

export const OAUTH_PROVIDERS: OAuthProvider[] = [
  {
    id: 'google',
    name: 'Google',
    color: '#4285F4',
    letter: 'G',
    fields: [
      { key: 'clientId', label: 'App ID', hint: 'Enter ID' },
      { key: 'clientSecret', label: 'App Secret', hint: 'Enter Secret', type: 'secret' },
    ],
    setupNote:
      "To complete the setup, create an OAuth2 client ID with 'Web application' as the application type, then add this redirect URI to your Google configuration.",
  },
  {
    id: 'github',
    name: 'GitHub',
    color: '#24292E',
    letter: '',
    fields: [
      { key: 'clientId', label: 'App ID', hint: 'Enter ID' },
      { key: 'clientSecret', label: 'App Secret', hint: 'Enter App Secret', type: 'secret' },
    ],
  },
  {
    id: 'apple',
    name: 'Apple',
    color: '#555555',
    letter: '',
    fields: [
      { key: 'serviceId', label: 'Services ID', hint: 'com.company.appname' },
      { key: 'keyId', label: 'Key ID', hint: 'SHAB13ROFN' },
      { key: 'teamId', label: 'Team ID', hint: 'ELA2CD3AED' },
      { key: 'p8', label: 'P8 File', hint: '-----BEGIN PRIVATE KEY-----\n...', type: 'multiline' },
    ],
  },
  {
    id: 'facebook',
    name: 'Facebook',
    color: '#1877F2',
    letter: 'f',
    fields: [
      { key: 'clientId', label: 'App ID', hint: 'Enter ID' },
      { key: 'clientSecret', label: 'App Secret', hint: 'Enter App Secret', type: 'secret' },
    ],
  },
  {
    id: 'discord',
    name: 'Discord',
    color: '#5865F2',
    letter: 'D',
    fields: [
      { key: 'clientId', label: 'Client ID', hint: 'Enter Client ID' },
      { key: 'clientSecret', label: 'Client Secret', hint: 'Enter Client Secret', type: 'secret' },
    ],
  },
  {
    id: 'twitter',
    name: 'Twitter / X',
    color: '#000000',
    letter: 'X',
    fields: [
      { key: 'clientId', label: 'Consumer Key', hint: 'Enter Consumer Key' },
      { key: 'clientSecret', label: 'Consumer Secret', hint: 'Enter Consumer Secret', type: 'secret' },
    ],
  },
  {
    id: 'microsoft',
    name: 'Microsoft',
    color: '#00A1F1',
    letter: 'M',
    fields: [
      { key: 'clientId', label: 'App (client) ID', hint: 'Enter Client ID' },
      { key: 'clientSecret', label: 'Client Secret Value', hint: 'Enter Client Secret', type: 'secret' },
      { key: 'tenantId', label: 'Tenant ID (optional)', hint: 'common' },
    ],
  },
  {
    id: 'slack',
    name: 'Slack',
    color: '#4A154B',
    letter: 'S',
    fields: [
      { key: 'clientId', label: 'Client ID', hint: 'Enter ID' },
      { key: 'clientSecret', label: 'Client Secret', hint: 'Enter Client Secret', type: 'secret' },
    ],
  },
  {
    id: 'spotify',
    name: 'Spotify',
    color: '#1DB954',
    letter: '',
    fields: [
      { key: 'clientId', label: 'Client ID', hint: 'Enter ID' },
      { key: 'clientSecret', label: 'Client Secret', hint: 'Enter Client Secret', type: 'secret' },
    ],
  },
  {
    id: 'linkedin',
    name: 'LinkedIn',
    color: '#0A66C2',
    letter: 'in',
    fields: [
      { key: 'clientId', label: 'Client ID', hint: 'Enter ID' },
      { key: 'clientSecret', label: 'Client Secret', hint: 'Enter Client Secret', type: 'secret' },
    ],
  },
  {
    id: 'gitlab',
    name: 'GitLab',
    color: '#FC6D26',
    letter: '',
    fields: [
      { key: 'clientId', label: 'App ID', hint: 'Enter ID' },
      { key: 'clientSecret', label: 'App Secret', hint: 'Enter App Secret', type: 'secret' },
    ],
  },
  {
    id: 'bitbucket',
    name: 'Bitbucket',
    color: '#0052CC',
    letter: 'B',
    fields: [
      { key: 'clientId', label: 'Key (Client ID)', hint: 'Enter Key' },
      { key: 'clientSecret', label: 'Secret (Client Secret)', hint: 'Enter Secret', type: 'secret' },
    ],
  },
  {
    id: 'twitch',
    name: 'Twitch',
    color: '#9146FF',
    letter: 'T',
    fields: [
      { key: 'clientId', label: 'Client ID', hint: 'Enter ID' },
      { key: 'clientSecret', label: 'Client Secret', hint: 'Enter Client Secret', type: 'secret' },
    ],
  },
  {
    id: 'notion',
    name: 'Notion',
    color: '#191919',
    letter: 'N',
    fields: [
      { key: 'clientId', label: 'OAuth Client ID', hint: 'Enter ID' },
      { key: 'clientSecret', label: 'OAuth Client Secret', hint: 'Enter Secret', type: 'secret' },
    ],
  },
  {
    id: 'stripe',
    name: 'Stripe',
    color: '#635BFF',
    letter: 'S',
    fields: [
      { key: 'clientId', label: 'Client ID', hint: 'ca_xxxxxxxxxxxx' },
      { key: 'clientSecret', label: 'Secret Key', hint: 'sk_live_xxxxxxxxxxxx', type: 'secret' },
    ],
  },
];
