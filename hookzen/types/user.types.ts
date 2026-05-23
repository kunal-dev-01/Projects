// =========================================================
// USER INTERFACES 
// =========================================================

export interface WebhookSettings {
    delivery_failures: boolean;
    endpoint_disabled: boolean;
    rate_limit_warnings: boolean;
    security_alerts: boolean;
}

export interface UserPreferences {
    notifications: WebhookSettings;
    language: string;
}

export interface UserProfile {
    id: string;
    email: string;
    email_verified: boolean;
    full_name: string;
    timezone: string;
    locale: string;
    two_factor_enabled: boolean;
    preferences: UserPreferences;
    created_at: string;
}